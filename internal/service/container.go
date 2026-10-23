package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type ContainerService struct {
	cli *client.Client
}

func NewContainerService() (*ContainerService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("无法连接 Docker: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker 守护进程无响应: %w", err)
	}
	return &ContainerService{cli: cli}, nil
}

func (s *ContainerService) Close() error {
	return s.cli.Close()
}

type ContainerInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Status  string   `json:"status"`
	State   string   `json:"state"`
	Created int64    `json:"created"`
	Ports   []string `json:"ports"`
	Managed bool     `json:"managed"`
}

// 获取所有容器
func (s *ContainerService) ListContainers(all bool) ([]ContainerInfo, error) {
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: all})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := c.ID[:12]
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := make([]string, 0, len(c.Ports))
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}
		result = append(result, ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: c.Created,
			Ports:   ports,
			Managed: c.Labels["managed-by"] == "upanel",
		})
	}
	return result, nil
}

// 启动容器
func (s *ContainerService) StartContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
}

// 停止容器
func (s *ContainerService) StopContainer(id string) error {
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return s.cli.ContainerStop(ctx, id, &timeout)
}

// 重启容器
func (s *ContainerService) RestartContainer(id string) error {
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.cli.ContainerRestart(ctx, id, &timeout)
}

// 删除容器
func (s *ContainerService) RemoveContainer(id string, force bool) error {
	options := types.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.cli.ContainerRemove(ctx, id, options)
}

// 获取容器日志（解复用 Docker 流）
func (s *ContainerService) GetContainerLogs(id string, tail int) (string, error) {
	if tail <= 0 || tail > 5000 {
		tail = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rc, err := s.cli.ContainerLogs(ctx, id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	})
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var out strings.Builder
	if _, err := copyDemux(&out, rc); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
