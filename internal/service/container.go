package service

import (
    "context"
    "fmt"
    "io"
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
        return nil, err
    }
    return &ContainerService{cli: cli}, nil
}

func (s *ContainerService) Close() error {
    return s.cli.Close()
}

type ContainerInfo struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Image   string `json:"image"`
    Status  string `json:"status"`
    State   string `json:"state"`
    Created int64  `json:"created"`
}

// 获取所有容器
func (s *ContainerService) ListContainers(all bool) ([]ContainerInfo, error) {
    containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: all})
    if err != nil {
        return nil, err
    }

    var result []ContainerInfo
    for _, c := range containers {
        name := strings.TrimPrefix(c.Names[0], "/")
        result = append(result, ContainerInfo{
            ID:      c.ID[:12],
            Name:    name,
            Image:   c.Image,
            Status:  c.Status,
            State:   c.State,
            Created: c.Created,
        })
    }
    return result, nil
}

// 启动容器
func (s *ContainerService) StartContainer(id string) error {
    return s.cli.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

// 停止容器
func (s *ContainerService) StopContainer(id string) error {
    timeout := 10 * time.Second
    return s.cli.ContainerStop(context.Background(), id, &timeout)
}

// 重启容器
func (s *ContainerService) RestartContainer(id string) error {
    timeout := 10 * time.Second
    return s.cli.ContainerRestart(context.Background(), id, &timeout)
}

// 删除容器
func (s *ContainerService) RemoveContainer(id string, force bool) error {
    options := types.ContainerRemoveOptions{
        Force:         force,
        RemoveVolumes: true,
    }
    return s.cli.ContainerRemove(context.Background(), id, options)
}

// 获取容器日志
func (s *ContainerService) GetContainerLogs(id string, tail int) (string, error) {
    options := types.ContainerLogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Tail:       fmt.Sprintf("%d", tail),
    }
    logs, err := s.cli.ContainerLogs(context.Background(), id, options)
    if err != nil {
        return "", err
    }
    defer logs.Close()

    buf := new(strings.Builder)
    _, err = io.Copy(buf, logs)
    return buf.String(), err
}
