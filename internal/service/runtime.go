package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// NetworkName 所有 UPanel 托管容器共用的 Docker 网络
const NetworkName = "upanel_network"

// Runtime 封装 Docker 操作。面板不再依赖宿主机上的 docker compose 二进制，
// 全部通过 Docker API 完成，避免 "命令不存在" 导致的部署失败。
type Runtime struct {
	cli *client.Client
}

// NewRuntime 创建 Docker 运行时客户端
func NewRuntime() (*Runtime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("无法连接 Docker，请确认 Docker 已安装且服务正在运行: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker 守护进程无响应，请确认 docker 服务已启动: %w", err)
	}
	return &Runtime{cli: cli}, nil
}

func (r *Runtime) Close() error {
	return r.cli.Close()
}

// Client 暴露底层客户端，供需要细粒度操作的场景使用
func (r *Runtime) Client() *client.Client {
	return r.cli
}

// EnsureNetwork 确保共享网络存在
func (r *Runtime) EnsureNetwork() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := r.cli.NetworkInspect(ctx, NetworkName, types.NetworkInspectOptions{}); err == nil {
		return nil
	}
	_, err := r.cli.NetworkCreate(ctx, NetworkName, types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{"managed-by": "upanel"},
	})
	if err != nil {
		return fmt.Errorf("创建网络 %s 失败: %w", NetworkName, err)
	}
	return nil
}

// ImageExists 判断镜像是否已存在于本地
func (r *Runtime) ImageExists(ref string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	list, err := r.cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false
	}
	for _, img := range list {
		for _, tag := range img.RepoTags {
			if tag == ref {
				return true
			}
		}
	}
	return false
}

// PullImage 拉取镜像；已存在时跳过。
// logFn 用于把拉取进度回传给调用方（前端安装日志）。
func (r *Runtime) PullImage(ref string, logFn func(string)) error {
	if r.ImageExists(ref) {
		if logFn != nil {
			logFn("镜像已存在，跳过拉取: " + ref)
		}
		return nil
	}
	if logFn != nil {
		logFn("开始拉取镜像: " + ref)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	rc, err := r.cli.ImagePull(ctx, ref, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("拉取镜像 %s 失败（可尝试在「设置」中配置镜像加速地址）: %w", ref, err)
	}
	defer rc.Close()

	// Docker ImagePull 在 HTTP 200 下也可能通过 JSON 流返回错误，
	// 必须解析流内容，否则只会得到后续 ContainerCreate 的 "No such image"。
	dec := json.NewDecoder(rc)
	lastLog := time.Now()
	for {
		var msg struct {
			Status      string `json:"status"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取镜像 %s 拉取进度失败: %w", ref, err)
		}
		if msg.ErrorDetail.Message != "" {
			return fmt.Errorf("拉取镜像 %s 失败: %s", ref, msg.ErrorDetail.Message)
		}
		if msg.Error != "" {
			return fmt.Errorf("拉取镜像 %s 失败: %s", ref, msg.Error)
		}
		if logFn != nil && msg.Status != "" && time.Since(lastLog) > 2*time.Second {
			logFn(fmt.Sprintf("拉取 %s: %s", ref, msg.Status))
			lastLog = time.Now()
		}
	}
	if logFn != nil {
		logFn("镜像拉取完成: " + ref)
	}
	return nil
}

// ContainerSpec 描述一个待创建的容器
type ContainerSpec struct {
	Name            string            // 容器名（唯一）
	Image           string            // 镜像
	Env             []string          // 环境变量
	Volumes         map[string]string // 宿主机路径 -> 容器内挂载点
	ReadOnlyVolumes []string          // 需要以只读方式挂载的宿主机路径（Volumes 的 key）
	Ports           map[string]string // 容器内端口 -> 宿主机端口（空字符串表示随机）
	Network         string            // 网络名，默认 upanel_network
	Labels          map[string]string // 额外容器标签
	Entrypoint      []string
	Cmd             []string
	MemoryMB        int64 // 内存限制，0 表示不限制
	// BindLocalhost 为 true 时端口只绑定 127.0.0.1。
	// 数据库、应用商店等带凭据的服务默认必须仅本机访问。
	BindLocalhost bool
}

// CreateAndStart 创建并启动容器。若同名容器已存在：
//   - 运行中 -> 直接返回
//   - 已停止 -> 删除后重建（保证配置变更生效）
func (r *Runtime) CreateAndStart(spec ContainerSpec) (string, error) {
	if err := r.EnsureNetwork(); err != nil {
		return "", err
	}

	netName := spec.Network
	if netName == "" {
		netName = NetworkName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 同名容器处理
	if info, err := r.FindContainer(spec.Name); err == nil {
		if info.State == "running" {
			return info.ID, nil
		}
		timeout := 5 * time.Second
		_ = r.cli.ContainerStop(ctx, info.ID, &timeout)
		_ = r.cli.ContainerRemove(ctx, info.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
	}

	// 卷挂载
	roSet := map[string]bool{}
	for _, p := range spec.ReadOnlyVolumes {
		roSet[p] = true
	}
	var binds []string
	for hostPath, containerPath := range spec.Volumes {
		if hostPath == "" || containerPath == "" {
			continue
		}
		// 注意：hostPath 可能是文件（如 nginx.conf）而不是目录。
		// 无条件 MkdirAll 会在文件已存在时返回 "not a directory"，导致网关无法创建。
		if _, statErr := os.Stat(hostPath); statErr != nil {
			if !os.IsNotExist(statErr) {
				return "", fmt.Errorf("检查挂载路径 %s 失败: %w", hostPath, statErr)
			}
			if err := os.MkdirAll(hostPath, 0755); err != nil {
				return "", fmt.Errorf("创建挂载目录 %s 失败: %w", hostPath, err)
			}
		}
		opts := make([]string, 0, 2)
		if roSet[hostPath] {
			opts = append(opts, "ro")
		}
		// SELinux 主机（CentOS/RHEL/Fedora）需要 :z 重新标记，否则容器内 nginx/php
		// 会因 permission denied 无法读取站点文件；docker.sock 不做 relabel。
		if !strings.HasSuffix(hostPath, "docker.sock") {
			opts = append(opts, "z")
		}
		mode := ""
		if len(opts) > 0 {
			mode = ":" + strings.Join(opts, ",")
		}
		binds = append(binds, fmt.Sprintf("%s:%s%s", hostPath, containerPath, mode))
	}

	// 端口映射
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	hostIP := "0.0.0.0"
	if spec.BindLocalhost {
		hostIP = "127.0.0.1"
	}
	for containerPort, hostPort := range spec.Ports {
		p, err := nat.ParsePortSpec(containerPort)
		if err != nil || len(p) == 0 {
			return "", fmt.Errorf("无效的容器端口: %s", containerPort)
		}
		exposedPorts[p[0].Port] = struct{}{}
		if hostPort != "" {
			portBindings[p[0].Port] = []nat.PortBinding{{HostIP: hostIP, HostPort: hostPort}}
		} else {
			portBindings[p[0].Port] = []nat.PortBinding{{HostIP: hostIP}}
		}
	}

	labels := map[string]string{"managed-by": "upanel"}
	for k, v := range spec.Labels {
		labels[k] = v
	}
	cfg := &container.Config{
		Image:        spec.Image,
		Env:          spec.Env,
		ExposedPorts: exposedPorts,
		Labels:       labels,
	}
	if len(spec.Entrypoint) > 0 {
		cfg.Entrypoint = spec.Entrypoint
	}
	if len(spec.Cmd) > 0 {
		cfg.Cmd = spec.Cmd
	}

	hostCfg := &container.HostConfig{
		Binds:         binds,
		PortBindings:  portBindings,
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	if spec.MemoryMB > 0 {
		hostCfg.Memory = spec.MemoryMB * 1024 * 1024
	}

	networkID := netName
	if n, err := r.cli.NetworkInspect(ctx, netName, types.NetworkInspectOptions{}); err == nil {
		networkID = n.ID
	}
	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {NetworkID: networkID},
		},
	}

	resp, err := r.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, spec.Name)
	if err != nil {
		return "", fmt.Errorf("创建容器 %s 失败: %w", spec.Name, err)
	}

	if err := r.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// 启动失败时清理，避免残留容器占用名字
		_ = r.cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("启动容器 %s 失败: %w", spec.Name, err)
	}

	return resp.ID, nil
}

// RuntimeContainer 容器状态快照
type RuntimeContainer struct {
	ID    string
	Name  string
	State string // running / exited / created
	Image string
}

// FindContainer 按名字查找容器（名字带或不带前导斜杠均可）
func (r *Runtime) FindContainer(name string) (*RuntimeContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	list, err := r.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	target := name
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	for _, c := range list {
		for _, n := range c.Names {
			if n == target {
				return &RuntimeContainer{
					ID:    c.ID,
					Name:  strings.TrimPrefix(n, "/"),
					State: c.State,
					Image: c.Image,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("容器 %s 不存在", name)
}

// Start 启动容器
func (r *Runtime) Start(name string) error {
	info, err := r.FindContainer(name)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return r.cli.ContainerStart(ctx, info.ID, types.ContainerStartOptions{})
}

// Stop 停止容器
func (r *Runtime) Stop(name string) error {
	info, err := r.FindContainer(name)
	if err != nil {
		return err
	}
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return r.cli.ContainerStop(ctx, info.ID, &timeout)
}

// Remove 强制删除容器
func (r *Runtime) Remove(name string) error {
	info, err := r.FindContainer(name)
	if err != nil {
		// 容器不存在视为已删除，保证幂等
		return nil
	}
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = r.cli.ContainerStop(ctx, info.ID, &timeout)
	return r.cli.ContainerRemove(ctx, info.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

// ExecResult 在容器内执行命令，返回合并后的输出与退出码。
// 使用 stdcopy 解复用 Docker 的多路复用流，避免日志里出现二进制头。
func (r *Runtime) ExecResult(name string, cmd ...string) (string, int, error) {
	info, err := r.FindContainer(name)
	if err != nil {
		return "", -1, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	idResp, err := r.cli.ContainerExecCreate(ctx, info.ID, types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", -1, fmt.Errorf("创建 exec 失败: %w", err)
	}
	resp, err := r.cli.ContainerExecAttach(ctx, idResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", -1, fmt.Errorf("attach exec 失败: %w", err)
	}
	defer resp.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, resp.Reader); err != nil {
		return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), -1, err
	}
	inspect, err := r.cli.ContainerExecInspect(ctx, idResp.ID)
	if err != nil {
		return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), -1, err
	}
	out := strings.TrimSpace(stdout.String())
	if s := strings.TrimSpace(stderr.String()); s != "" {
		if out != "" {
			out += "\n"
		}
		out += s
	}
	return out, inspect.ExitCode, nil
}

// Exec 在容器内执行命令；非零退出码返回错误并附带输出。
func (r *Runtime) Exec(name string, cmd ...string) (string, error) {
	out, code, err := r.ExecResult(name, cmd...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, fmt.Errorf("命令退出码 %d: %s", code, out)
	}
	return out, nil
}

// ContainerState 返回容器状态，不存在返回 ""
func (r *Runtime) ContainerState(name string) string {
	info, err := r.FindContainer(name)
	if err != nil {
		return ""
	}
	return info.State
}

// ListManaged 列出所有由 UPanel 管理的容器名
func (r *Runtime) ListManaged() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	args := filters.NewArgs()
	args.Add("label", "managed-by=upanel")
	list, err := r.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: args})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range list {
		if len(c.Names) > 0 {
			names = append(names, strings.TrimPrefix(c.Names[0], "/"))
		}
	}
	return names, nil
}

// UsedHostPorts 收集已被容器占用的宿主机端口
func (r *Runtime) UsedHostPorts() map[int]bool {
	used := map[int]bool{}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	list, err := r.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return used
	}
	for _, c := range list {
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				used[int(p.PublicPort)] = true
			}
		}
	}
	return used
}

// HostPortOccupied 判断宿主机端口是否被容器占用
func (r *Runtime) HostPortOccupied(port int) bool {
	if port <= 0 {
		return false
	}
	if r.UsedHostPorts()[port] {
		return true
	}
	// 同时检查主机上是否已有服务监听
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

// FindFreePort 从 start 开始寻找一个未被占用的端口
func (r *Runtime) FindFreePort(start int) int {
	if start <= 0 {
		start = 8000
	}
	for p := start; p < start+500; p++ {
		if !r.HostPortOccupied(p) {
			return p
		}
	}
	return 0
}

// HostPort 查询容器某个容器端口映射到宿主机的端口
func (r *Runtime) HostPort(name, containerPort string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	list, err := r.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return 0
	}
	target := "/" + name
	for _, c := range list {
		for _, n := range c.Names {
			if n != target {
				continue
			}
			for _, p := range c.Ports {
				if fmt.Sprintf("%d", p.PrivatePort) == containerPort && p.PublicPort > 0 {
					return int(p.PublicPort)
				}
			}
		}
	}
	return 0
}

// ImageInfo 镜像快照
type ImageInfo struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

// ListImages 列出本地镜像（用于容器页「镜像」标签）
func (r *Runtime) ListImages() ([]ImageInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	list, err := r.cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]ImageInfo, 0, len(list))
	for _, img := range list {
		tags := img.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}
		id := img.ID
		if len(id) > 19 {
			id = id[7:19]
		}
		out = append(out, ImageInfo{ID: id, Tags: tags, Size: img.Size, Created: img.Created})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, nil
}

// VolumeInfo 数据卷快照
type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"createdAt"`
}

// ListVolumes 列出数据卷
func (r *Runtime) ListVolumes() ([]VolumeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := r.cli.VolumeList(ctx, filters.NewArgs())
	if err != nil {
		return nil, err
	}
	out := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		out = append(out, VolumeInfo{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, CreatedAt: v.CreatedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// NetworkInfo 网络快照
type NetworkInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

// ListNetworks 列出 Docker 网络
func (r *Runtime) ListNetworks() ([]NetworkInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	list, err := r.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]NetworkInfo, 0, len(list))
	for _, n := range list {
		out = append(out, NetworkInfo{ID: n.ID, Name: n.Name, Driver: n.Driver, Scope: n.Scope})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Logs 读取容器日志（自动解复用 Docker 流），最多返回 2MB。
func (r *Runtime) Logs(name string, tail int) (string, error) {
	info, err := r.FindContainer(name)
	if err != nil {
		return "", err
	}
	if tail <= 0 || tail > 5000 {
		tail = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rc, err := r.cli.ContainerLogs(ctx, info.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	})
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, rc); err != nil {
		return stdout.String() + stderr.String(), err
	}
	out := stdout.String()
	if s := stderr.String(); s != "" {
		if out != "" {
			out += "\n"
		}
		out += s
	}
	const maxLen = 2 << 20
	if len(out) > maxLen {
		out = out[len(out)-maxLen:]
	}
	return out, nil
}

// copyDemux 解复用 Docker 容器的多路复用输出流（stdout+stderr）。
func copyDemux(dst *strings.Builder, src io.Reader) (int64, error) {
	var stdout, stderr bytes.Buffer
	n, err := stdcopy.StdCopy(&stdout, &stderr, src)
	if stdout.Len() > 0 {
		dst.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if dst.Len() > 0 && !strings.HasSuffix(dst.String(), "\n") {
			dst.WriteString("\n")
		}
		dst.WriteString(stderr.String())
	}
	return n, err
}
