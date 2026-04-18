package service

import (
	"context"
	"runtime"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/load"
)

type SystemInfo struct {
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	CPU          int       `json:"cpu"`
	GoVersion    string    `json:"go_version"`
	Uptime       string    `json:"uptime"`
	StartTime    time.Time `json:"start_time"`
	
	// 系统资源
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryPercent float64  `json:"memory_percent"`
	MemoryUsed   uint64    `json:"memory_used"`
	MemoryTotal  uint64    `json:"memory_total"`
	DiskPercent  float64   `json:"disk_percent"`
	DiskUsed     uint64    `json:"disk_used"`
	DiskTotal    uint64    `json:"disk_total"`
	LoadAvg      float64   `json:"load_avg"`
	
	// Docker 信息
	DockerVersion string `json:"docker_version"`
	Containers    int    `json:"containers"`
	Running       int    `json:"running"`
	Stopped       int    `json:"stopped"`
	Images        int    `json:"images"`
}

type SystemService struct {
	cli        *client.Client
	startTime  time.Time
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

// 获取系统信息
func (s *SystemService) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPU:       runtime.NumCPU(),
		GoVersion: runtime.Version(),
		StartTime: s.startTime,
		Uptime:    formatUptime(time.Since(s.startTime)),
	}

	// 获取 CPU 使用率
	cpuPercent, _ := cpu.Percent(0, false)
	if len(cpuPercent) > 0 {
		info.CPUPercent = cpuPercent[0]
	}

	// 获取内存信息
	memInfo, _ := mem.VirtualMemory()
	if memInfo != nil {
		info.MemoryPercent = memInfo.UsedPercent
		info.MemoryUsed = memInfo.Used
		info.MemoryTotal = memInfo.Total
	}

	// 获取磁盘信息（根目录）
	diskInfo, _ := disk.Usage("/")
	if diskInfo != nil {
		info.DiskPercent = diskInfo.UsedPercent
		info.DiskUsed = diskInfo.Used
		info.DiskTotal = diskInfo.Total
	}

	// 获取系统负载
	loadAvg, _ := load.Avg()
	if loadAvg != nil {
		info.LoadAvg = loadAvg.Load1
	}

	// 获取 Docker 信息
	dockerInfo, err := s.cli.Info(context.Background())
	if err == nil {
		info.DockerVersion = dockerInfo.ServerVersion
		info.Containers = dockerInfo.Containers
		info.Running = dockerInfo.ContainersRunning
		info.Stopped = dockerInfo.ContainersStopped
		info.Images = dockerInfo.Images
	}

	return info, nil
}

// 格式化运行时间
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	
	if days > 0 {
		if hours == 0 {
			if days == 1 {
				return "1 天 " + formatMinutes(minutes)
			}
			return formatDaysCount(days) + " 天 " + formatMinutes(minutes)
		}
		if minutes == 0 {
			return formatDaysCount(days) + " 天 " + formatHoursCount(hours) + " 小时"
		}
		return formatDaysCount(days) + " 天 " + formatHoursCount(hours) + " 小时 " + formatMinutesCount(minutes) + " 分钟"
	}
	if hours > 0 {
		if minutes == 0 {
			return formatHoursCount(hours) + " 小时"
		}
		return formatHoursCount(hours) + " 小时 " + formatMinutesCount(minutes) + " 分钟"
	}
	return formatMinutesCount(minutes) + " 分钟"
}

func formatDaysCount(days int) string {
	return string(rune(days))
}

func formatHoursCount(hours int) string {
	return string(rune(hours))
}

func formatMinutesCount(minutes int) string {
	return string(rune(minutes))
}

func formatMinutes(minutes int) string {
	if minutes == 0 {
		return ""
	}
	return formatMinutesCount(minutes) + " 分钟"
}
