package service

import (
	"context"
	"runtime"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
)

type SystemInfo struct {
	Hostname    string    `json:"hostname"`
	OS          string    `json:"os"`
	Platform    string    `json:"platform"`
	Arch        string    `json:"arch"`
	Kernel      string    `json:"kernel"`
	CPU         int       `json:"cpu"`
	GoVersion   string    `json:"go_version"`
	Uptime      string    `json:"uptime"`
	PanelUptime string    `json:"panel_uptime"`
	StartTime   time.Time `json:"start_time"`

	// 系统资源
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	SwapPercent   float64 `json:"swap_percent"`
	SwapUsed      uint64  `json:"swap_used"`
	SwapTotal     uint64  `json:"swap_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	LoadAvg       float64 `json:"load_avg"`
	Load1         float64 `json:"load1"`
	Load5         float64 `json:"load5"`
	Load15        float64 `json:"load15"`

	// 网络累计字节数（前端两次采样计算速率）
	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`

	// Docker 信息
	DockerVersion string `json:"docker_version"`
	DockerStatus  string `json:"docker_status"`
	Containers    int    `json:"containers"`
	Running       int    `json:"running"`
	Stopped       int    `json:"stopped"`
	Images        int    `json:"images"`
}

type SystemService struct {
	cli       *client.Client
	startTime time.Time
}

func NewSystemService() (*SystemService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &SystemService{
		cli:       cli,
		startTime: time.Now(),
	}, nil
}

func (s *SystemService) Close() error {
	return s.cli.Close()
}

// GetSystemInfo 获取系统信息
func (s *SystemService) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPU:         runtime.NumCPU(),
		GoVersion:   runtime.Version(),
		StartTime:   s.startTime,
		PanelUptime: formatDuration(time.Since(s.startTime)),
	}

	if hi, err := host.Info(); err == nil && hi != nil {
		info.Hostname = hi.Hostname
		info.Platform = hi.Platform + " " + hi.PlatformVersion
		info.Kernel = hi.KernelArch + " " + hi.KernelVersion
		info.Uptime = formatDuration(time.Duration(hi.Uptime) * time.Second)
	}

	cpuPercent, _ := cpu.Percent(0, false)
	if len(cpuPercent) > 0 {
		info.CPUPercent = round2(cpuPercent[0])
	}

	if memInfo, _ := mem.VirtualMemory(); memInfo != nil {
		info.MemoryPercent = round2(memInfo.UsedPercent)
		info.MemoryUsed = memInfo.Used
		info.MemoryTotal = memInfo.Total
	}
	if swap, _ := mem.SwapMemory(); swap != nil {
		info.SwapPercent = round2(swap.UsedPercent)
		info.SwapUsed = swap.Used
		info.SwapTotal = swap.Total
	}

	if diskInfo, _ := disk.Usage("/"); diskInfo != nil {
		info.DiskPercent = round2(diskInfo.UsedPercent)
		info.DiskUsed = diskInfo.Used
		info.DiskTotal = diskInfo.Total
	}

	if loadAvg, _ := load.Avg(); loadAvg != nil {
		info.LoadAvg = round2(loadAvg.Load1)
		info.Load1 = round2(loadAvg.Load1)
		info.Load5 = round2(loadAvg.Load5)
		info.Load15 = round2(loadAvg.Load15)
	}

	if counters, err := gnet.IOCounters(false); err == nil && len(counters) > 0 {
		for _, c := range counters {
			info.NetworkRxBytes += c.BytesRecv
			info.NetworkTxBytes += c.BytesSent
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dockerInfo, err := s.cli.Info(ctx)
	if err == nil {
		info.DockerStatus = "running"
		info.DockerVersion = dockerInfo.ServerVersion
		info.Containers = dockerInfo.Containers
		info.Running = dockerInfo.ContainersRunning
		info.Stopped = dockerInfo.ContainersStopped
		info.Images = dockerInfo.Images
	} else {
		info.DockerStatus = "unavailable"
	}

	return info, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// formatDuration 把时长格式化为 "x 天 y 小时 z 分钟"
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return strconv.Itoa(days) + " 天 " + strconv.Itoa(hours) + " 小时 " + strconv.Itoa(minutes) + " 分钟"
	case hours > 0:
		return strconv.Itoa(hours) + " 小时 " + strconv.Itoa(minutes) + " 分钟"
	default:
		return strconv.Itoa(minutes) + " 分钟"
	}
}
