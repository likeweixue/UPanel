package service

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

type InstallLogger struct {
	mu   sync.RWMutex
	logs map[string]*bytes.Buffer
}

var installLogger = &InstallLogger{
	logs: make(map[string]*bytes.Buffer),
}

func GetInstallLogger() *InstallLogger {
	return installLogger
}

func (l *InstallLogger) StartLog(taskId string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs[taskId] = &bytes.Buffer{}
	l.logs[taskId].WriteString(fmt.Sprintf("[%s] 安装任务开始\n", getTimestamp()))
}

func (l *InstallLogger) WriteLog(taskId, msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if buf, ok := l.logs[taskId]; ok {
		buf.WriteString(fmt.Sprintf("[%s] %s\n", getTimestamp(), msg))
	}
}

func (l *InstallLogger) GetLogs(taskId string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if buf, ok := l.logs[taskId]; ok {
		return buf.String()
	}
	return ""
}

func (l *InstallLogger) EndLog(taskId string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if buf, ok := l.logs[taskId]; ok {
		buf.WriteString(fmt.Sprintf("[%s] 安装任务完成\n", getTimestamp()))
	}
}

func (l *InstallLogger) ErrorLog(taskId, errMsg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if buf, ok := l.logs[taskId]; ok {
		buf.WriteString(fmt.Sprintf("[%s] ❌ 错误: %s\n", getTimestamp(), errMsg))
	}
}

func (l *InstallLogger) RemoveLog(taskId string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.logs, taskId)
}

func getTimestamp() string {
	return time.Now().Format("2006/01/02 15:04:05")
}
