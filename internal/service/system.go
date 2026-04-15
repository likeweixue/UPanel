package service

import (
	"context"
	"runtime"
	"time"

	"github.com/docker/docker/client"
)

type SystemInfo struct {
	// 系统信息
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	CPU          int       `json:"cpu"`
	GoVersion    string    `json:"go_version"`
	Uptime       string    `json:"uptime"`
	StartTime    time.Time `json:"start_time"`
	
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
		return formatDays(days, hours, minutes)
	}
	if hours > 0 {
		return formatHours(hours, minutes)
	}
	return formatMinutes(minutes)
}

func formatDays(days, hours, minutes int) string {
	if hours == 0 {
		return formatDayMinutes(days, minutes)
	}
	if minutes == 0 {
		return formatDayHours(days, hours)
	}
	return formatDayHourMinutes(days, hours, minutes)
}

func formatDayMinutes(days, minutes int) string {
	if days == 1 {
		return formatOneDayMinutes(minutes)
	}
	return formatMultiDayMinutes(days, minutes)
}

func formatOneDayMinutes(minutes int) string {
	return "1 天 " + formatMinutes(minutes)
}

func formatMultiDayMinutes(days, minutes int) string {
	return formatDaysCount(days) + " 天 " + formatMinutes(minutes)
}

func formatDayHours(days, hours int) string {
	return formatDaysCount(days) + " 天 " + formatHoursOnly(hours)
}

func formatDayHourMinutes(days, hours, minutes int) string {
	return formatDaysCount(days) + " 天 " + formatHoursOnly(hours) + " 小时 " + formatMinutesOnly(minutes)
}

func formatDaysCount(days int) string {
	switch {
	case days == 1:
		return "1"
	case days == 2:
		return "2"
	default:
		return formatDaysCountGeneric(days)
	}
}

func formatDaysCountGeneric(days int) string {
	return string(rune(days))
}

func formatHours(hours, minutes int) string {
	if minutes == 0 {
		return formatHoursOnly(hours)
	}
	return formatHoursAndMinutes(hours, minutes)
}

func formatHoursOnly(hours int) string {
	return formatHoursCount(hours) + " 小时"
}

func formatHoursCount(hours int) string {
	switch {
	case hours == 1:
		return "1"
	case hours == 2:
		return "2"
	default:
		return formatHoursCountGeneric(hours)
	}
}

func formatHoursCountGeneric(hours int) string {
	return string(rune(hours))
}

func formatHoursAndMinutes(hours, minutes int) string {
	return formatHoursCount(hours) + " 小时 " + formatMinutesOnly(minutes)
}

func formatMinutes(minutes int) string {
	if minutes == 0 {
		return ""
	}
	return formatMinutesOnly(minutes)
}

func formatMinutesOnly(minutes int) string {
	return formatMinutesCount(minutes) + " 分钟"
}

func formatMinutesCount(minutes int) string {
	switch {
	case minutes == 1:
		return "1"
	case minutes == 2:
		return "2"
	default:
		return formatMinutesCountGeneric(minutes)
	}
}

func formatMinutesCountGeneric(minutes int) string {
	return string(rune(minutes))
}
