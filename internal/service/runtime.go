package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
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
	ctx := context.Background()
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

	// 边读边丢弃即可，Docker 的进度 JSON 仅用于避免流阻塞
	buf := make([]byte, 32*1024)
	total := 0
	for {
		n, err := rc.Read(buf)
		total += n
		if logFn != nil && total > 0 && total%(1024*1024) < len(buf) {
			logFn(fmt.Sprintf("正在拉取 %s ... (%d KB)", ref, total/1024))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("拉取镜像 %s 中断: %w", ref, err)
		}
	}
	if logFn != nil {
		logFn("镜像拉取完成: " + ref)
	}
	return nil
}

// ContainerSpec 描述一个待创建的容器
type ContainerSpec struct {
	Name       string            // 容器名（唯一）
	Image      string            // 镜像
	Env        []string          // 环境变量
	Volumes    map[string]string // 宿主机路径 -> 容器内挂载点
	Ports      map[string]string // 容器内端口 -> 宿主机端口（空字符串表示随机）
	Network    string            // 网络名，默认 upanel_network
	Entrypoint []string
	Cmd        []string
	MemoryMB   int64 // 内存限制，0 表示不限制
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

	ctx := context.Background()

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
	var binds []string
	for hostPath, containerPath := range spec.Volumes {
		if hostPath == "" || containerPath == "" {
			continue
		}
		if err := os.MkdirAll(hostPath, 0755); err != nil {
			return "", fmt.Errorf("创建挂载目录 %s 失败: %w", hostPath, err)
		}
		binds = append(binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// 端口映射
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for containerPort, hostPort := range spec.Ports {
		p := nat.Port(containerPort)
		exposedPorts[p] = struct{}{}
		if hostPort != "" {
			portBindings[p] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}}
		} else {
			portBindings[p] = []nat.PortBinding{{HostIP: "0.0.0.0"}}
		}
	}

	cfg := &container.Config{
		Image:        spec.Image,
		Env:          spec.Env,
		ExposedPorts: exposedPorts,
		Labels:       map[string]string{"managed-by": "upanel"},
	}
	if len(spec.Entrypoint) > 0 {
		cfg.Entrypoint = spec.Entrypoint
	}
	if len(spec.Cmd) > 0 {
		cfg.Cmd = spec.Cmd
	}

	hostCfg := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	if spec.MemoryMB > 0 {
		hostCfg.Memory = spec.MemoryMB * 1024 * 1024
	}

	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {NetworkID: netName},
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
	return r.cli.ContainerStart(context.Background(), info.ID, types.ContainerStartOptions{})
}

// Stop 停止容器
func (r *Runtime) Stop(name string) error {
	info, err := r.FindContainer(name)
	if err != nil {
		return err
	}
	timeout := 10 * time.Second
	return r.cli.ContainerStop(context.Background(), info.ID, &timeout)
}

// Remove 强制删除容器
func (r *Runtime) Remove(name string) error {
	info, err := r.FindContainer(name)
	if err != nil {
		// 容器不存在视为已删除，保证幂等
		return nil
	}
	timeout := 5 * time.Second
	_ = r.cli.ContainerStop(context.Background(), info.ID, &timeout)
	return r.cli.ContainerRemove(context.Background(), info.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

// Exec 在容器内执行命令并返回输出
func (r *Runtime) Exec(name string, cmd ...string) (string, error) {
	info, err := r.FindContainer(name)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	idResp, err := r.cli.ContainerExecCreate(ctx, info.ID, types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("创建 exec 失败: %w", err)
	}
	resp, err := r.cli.ContainerExecAttach(ctx, idResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("attach exec 失败: %w", err)
	}
	defer resp.Close()

	out, err := io.ReadAll(resp.Reader)
	if err != nil {
		return string(out), err
	}
	return string(out), nil
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
