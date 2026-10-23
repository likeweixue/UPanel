package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"upanel/internal/config"
)

// CronJob 计划任务。命令以 root 权限执行，请只添加可信内容。
type CronJob struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Schedule   string     `json:"schedule"`
	Command    string     `json:"command"`
	Enabled    bool       `json:"enabled"`
	LastRun    *time.Time `json:"lastRun,omitempty"`
	LastStatus string     `json:"lastStatus"`
	LastOutput string     `json:"lastOutput"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// CronJobRequest 创建/更新计划任务
type CronJobRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
}

var (
	cronMu      sync.Mutex
	cronRunning = map[string]bool{}
	cronOnce    sync.Once
)

func cronFilePath() string {
	return filepath.Join(config.Current().DataPath, "cron.json")
}

func loadCronJobs() ([]CronJob, error) {
	data, err := os.ReadFile(cronFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []CronJob{}, nil
		}
		return nil, err
	}
	var jobs []CronJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("计划任务文件损坏: %w", err)
	}
	if jobs == nil {
		jobs = []CronJob{}
	}
	return jobs, nil
}

func saveCronJobs(jobs []CronJob) error {
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cronFilePath()), 0755); err != nil {
		return err
	}
	return writeFileAtomic(cronFilePath(), data, 0600)
}

// ListCronJobs 列出全部计划任务
func ListCronJobs() ([]CronJob, error) {
	cronMu.Lock()
	defer cronMu.Unlock()
	jobs, err := loadCronJobs()
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.Before(jobs[j].CreatedAt) })
	for i := range jobs {
		if cronRunning[jobs[i].ID] {
			jobs[i].LastStatus = "running"
		}
	}
	return jobs, nil
}

func validateCronRequest(req *CronJobRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Schedule = strings.TrimSpace(req.Schedule)
	req.Command = strings.TrimSpace(req.Command)
	if req.Name == "" {
		return fmt.Errorf("任务名称不能为空")
	}
	if len(req.Name) > 64 {
		return fmt.Errorf("任务名称最多 64 个字符")
	}
	if req.Command == "" {
		return fmt.Errorf("命令不能为空")
	}
	if len(req.Command) > 4096 {
		return fmt.Errorf("命令最多 4096 个字符")
	}
	if _, err := parseCron(req.Schedule); err != nil {
		return err
	}
	return nil
}

// CreateCronJob 创建计划任务
func CreateCronJob(req *CronJobRequest) (*CronJob, error) {
	if err := validateCronRequest(req); err != nil {
		return nil, err
	}
	cronMu.Lock()
	defer cronMu.Unlock()
	jobs, err := loadCronJobs()
	if err != nil {
		return nil, err
	}
	if len(jobs) >= 100 {
		return nil, fmt.Errorf("计划任务最多 100 个")
	}
	job := CronJob{
		ID:        randomID(),
		Name:      req.Name,
		Schedule:  req.Schedule,
		Command:   req.Command,
		Enabled:   req.Enabled,
		CreatedAt: time.Now(),
	}
	jobs = append(jobs, job)
	if err := saveCronJobs(jobs); err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateCronJob 更新计划任务
func UpdateCronJob(id string, req *CronJobRequest) (*CronJob, error) {
	if err := validateCronRequest(req); err != nil {
		return nil, err
	}
	cronMu.Lock()
	defer cronMu.Unlock()
	jobs, err := loadCronJobs()
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].ID != id {
			continue
		}
		jobs[i].Name = req.Name
		jobs[i].Schedule = req.Schedule
		jobs[i].Command = req.Command
		jobs[i].Enabled = req.Enabled
		if err := saveCronJobs(jobs); err != nil {
			return nil, err
		}
		return &jobs[i], nil
	}
	return nil, fmt.Errorf("计划任务不存在")
}

// DeleteCronJob 删除计划任务
func DeleteCronJob(id string) error {
	cronMu.Lock()
	defer cronMu.Unlock()
	jobs, err := loadCronJobs()
	if err != nil {
		return err
	}
	out := jobs[:0]
	found := false
	for _, j := range jobs {
		if j.ID == id {
			found = true
			continue
		}
		out = append(out, j)
	}
	if !found {
		return fmt.Errorf("计划任务不存在")
	}
	return saveCronJobs(out)
}

// RunCronJob 立即执行一次（异步）
func RunCronJob(id string) error {
	cronMu.Lock()
	jobs, err := loadCronJobs()
	if err != nil {
		cronMu.Unlock()
		return err
	}
	var target *CronJob
	for i := range jobs {
		if jobs[i].ID == id {
			cp := jobs[i]
			target = &cp
			break
		}
	}
	if target == nil {
		cronMu.Unlock()
		return fmt.Errorf("计划任务不存在")
	}
	if cronRunning[id] {
		cronMu.Unlock()
		return fmt.Errorf("任务正在执行中")
	}
	cronRunning[id] = true
	cronMu.Unlock()

	go func() {
		runCronJob(target)
		cronMu.Lock()
		delete(cronRunning, id)
		cronMu.Unlock()
	}()
	return nil
}

// StartCronScheduler 启动后台调度器（幂等）
func StartCronScheduler() {
	cronOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				cronTick()
			}
		}()
	})
}

func cronTick() {
	cronMu.Lock()
	jobs, err := loadCronJobs()
	if err != nil {
		cronMu.Unlock()
		return
	}
	now := time.Now()
	var due []CronJob
	for _, job := range jobs {
		if !job.Enabled || cronRunning[job.ID] {
			continue
		}
		if job.LastStatus == "running" {
			continue
		}
		spec, err := parseCron(job.Schedule)
		if err != nil || !spec.matches(now) {
			continue
		}
		if job.LastRun != nil && sameMinute(*job.LastRun, now) {
			continue
		}
		cronRunning[job.ID] = true
		due = append(due, job)
	}
	cronMu.Unlock()

	for i := range due {
		job := due[i]
		go func() {
			runCronJob(&job)
			cronMu.Lock()
			delete(cronRunning, job.ID)
			cronMu.Unlock()
		}()
	}
}

func runCronJob(job *CronJob) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", job.Command)
	// 不继承面板进程环境，避免 JWT_SECRET 等敏感变量泄露给任务命令
	cmd.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/root",
		"UPANEL_CRON=1",
	}
	out, err := cmd.CombinedOutput()
	status := "success"
	if err != nil {
		status = "failed"
		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
			out = append(out, []byte("\n[upanel] 命令执行超时（10 分钟）")...)
		} else {
			out = append(out, []byte("\n[upanel] "+err.Error())...)
		}
	}
	const maxOut = 16 << 10
	if len(out) > maxOut {
		out = out[len(out)-maxOut:]
	}

	cronMu.Lock()
	defer cronMu.Unlock()
	jobs, err := loadCronJobs()
	if err != nil {
		return
	}
	for i := range jobs {
		if jobs[i].ID != job.ID {
			continue
		}
		t := start
		jobs[i].LastRun = &t
		jobs[i].LastStatus = status
		jobs[i].LastOutput = string(out)
		_ = saveCronJobs(jobs)
		return
	}
}

func sameMinute(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd && a.Hour() == b.Hour() && a.Minute() == b.Minute()
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

// ---------------- cron 表达式 ----------------

type cronSpec struct {
	minute, hour, dom, month, dow map[int]bool
	domRestricted, dowRestricted  bool
}

func parseCron(expr string) (*cronSpec, error) {
	expr = strings.TrimSpace(expr)
	aliases := map[string]string{
		"@hourly":   "0 * * * *",
		"@daily":    "0 0 * * *",
		"@midnight": "0 0 * * *",
		"@weekly":   "0 0 * * 0",
		"@monthly":  "0 0 1 * *",
		"@yearly":   "0 0 1 1 *",
		"@annually": "0 0 1 1 *",
	}
	if v, ok := aliases[strings.ToLower(expr)]; ok {
		expr = v
	}
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron 表达式必须是 5 段（分 时 日 月 周），例如 */5 * * * *")
	}
	minute, _, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("分钟字段无效: %w", err)
	}
	hour, _, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("小时字段无效: %w", err)
	}
	dom, domR, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("日期字段无效: %w", err)
	}
	month, _, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("月份字段无效: %w", err)
	}
	dow, dowR, err := parseCronField(fields[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("星期字段无效: %w", err)
	}
	if dow[7] {
		dow[0] = true
	}
	return &cronSpec{
		minute: minute, hour: hour, dom: dom, month: month, dow: dow,
		domRestricted: domR, dowRestricted: dowR,
	}, nil
}

func parseCronField(field string, min, max int) (map[int]bool, bool, error) {
	out := map[int]bool{}
	field = strings.TrimSpace(field)
	if field == "" {
		return nil, false, fmt.Errorf("字段为空")
	}
	// 支持 7 表示周日（仅 dow）
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false, fmt.Errorf("包含空的列表项")
		}
		step := 1
		base := part
		if i := strings.Index(part, "/"); i >= 0 {
			base = part[:i]
			s, err := strconv.Atoi(part[i+1:])
			if err != nil || s < 1 || s > 100 {
				return nil, false, fmt.Errorf("步长无效: %s", part)
			}
			step = s
		}
		start, end := min, max
		if base == "*" {
			// 默认范围
		} else if i := strings.Index(base, "-"); i >= 0 {
			s, err1 := strconv.Atoi(base[:i])
			e, err2 := strconv.Atoi(base[i+1:])
			if err1 != nil || err2 != nil || s < min || e > max || s > e {
				return nil, false, fmt.Errorf("范围无效: %s", part)
			}
			start, end = s, e
		} else {
			s, err := strconv.Atoi(base)
			if err != nil || s < min || s > max {
				return nil, false, fmt.Errorf("数值无效: %s", part)
			}
			if strings.Contains(part, "/") {
				start, end = s, max
			} else {
				start, end = s, s
			}
		}
		for v := start; v <= end; v += step {
			out[v] = true
		}
	}
	if len(out) == 0 {
		return nil, false, fmt.Errorf("没有匹配的值")
	}
	restricted := strings.TrimSpace(field) != "*"
	return out, restricted, nil
}

func (c *cronSpec) matches(t time.Time) bool {
	if !c.minute[t.Minute()] || !c.hour[t.Hour()] || !c.month[int(t.Month())] {
		return false
	}
	domMatch := c.dom[t.Day()]
	dowMatch := c.dow[int(t.Weekday())]
	if c.domRestricted && c.dowRestricted {
		return domMatch || dowMatch
	}
	return domMatch && dowMatch
}
