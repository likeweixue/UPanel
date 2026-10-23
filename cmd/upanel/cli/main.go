package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"upanel/internal/config"
	"upanel/internal/service"
)

// Version 与面板主程序保持一致
var Version = "v0.1.7"

const serviceName = "upanel"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		return
	}

	// 允许 "up 9 password xxx" 这种连续参数形式
	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "0", "status":
		checkStatus()
	case "1", "start":
		ctl("start")
	case "2", "stop":
		ctl("stop")
	case "3", "restart":
		ctl("restart")
	case "5", "uninstall":
		uninstall()
	case "6", "user":
		getUserInfo()
	case "7", "listen":
		changeListenIP()
	case "8", "version", "-v", "--version":
		showVersion()
	case "9", "modify":
		sub := ""
		val := ""
		if len(rest) > 0 {
			sub = rest[0]
		}
		if len(rest) > 1 {
			val = rest[1]
		}
		modifySystem(sub, val)
	case "10", "reset":
		resetSystem()
	case "11", "restore":
		restoreData()
	case "gateway":
		gatewayStatus()
	case "info":
		panelInfo()
	case "logs":
		showLogs()
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("                    UPanel 命令行工具")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  up 0 | status      检查 UPanel 服务状态")
	fmt.Println("  up 1 | start       启动 UPanel 服务")
	fmt.Println("  up 2 | stop        停止 UPanel 服务")
	fmt.Println("  up 3 | restart     重启 UPanel 服务")
	fmt.Println("  up 5 | uninstall   卸载 UPanel（保留或删除数据可选）")
	fmt.Println("  up 6 | user        查看面板地址 / 用户名 / 端口")
	fmt.Println("  up 7 | listen      切换面板监听 IP")
	fmt.Println("  up 8 | version     查看版本")
	fmt.Println("  up 9 | modify      修改面板配置")
	fmt.Println("  up 10 | reset      重置面板密码与端口为默认值")
	fmt.Println("  up 11 | restore    恢复面板服务及数据")
	fmt.Println("  up gateway         查看网站网关状态")
	fmt.Println("  up info            查看安装路径等详细信息")
	fmt.Println("  up logs            查看面板运行日志")
	fmt.Println("")
	fmt.Println("修改配置 (up 9):")
	fmt.Println("  up 9 username <用户名>      修改面板用户名")
	fmt.Println("  up 9 password <新密码>      修改面板密码")
	fmt.Println("  up 9 port <端口>            修改面板端口")
	fmt.Println("  up 9 entry <入口>          修改安全入口（清空用 -）")
	fmt.Println("")
}

// baseDir 推断安装目录：优先环境变量 -> 可执行文件上级目录 -> /opt/upanel
func baseDir() string {
	if v := os.Getenv("UPANEL_BASE_DIR"); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "bin" {
			return filepath.Dir(dir)
		}
		return dir
	}
	return "/opt/upanel"
}

func loadCfg() *config.Config {
	os.Setenv("UPANEL_BASE_DIR", baseDir())
	if err := config.LoadConfig(); err != nil {
		fmt.Println("警告: 配置加载失败，将使用默认值:", err)
	}
	return config.Current()
}

func systemctlAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func ctl(action string) {
	if !systemctlAvailable() {
		fmt.Println("未检测到 systemd，请手动管理面板进程")
		return
	}
	fmt.Printf("正在%s UPanel ...\n", actionName(action))
	out, err := exec.Command("systemctl", action, serviceName).CombinedOutput()
	if err != nil {
		fmt.Printf("✗ 操作失败: %v\n%s\n", err, string(out))
		return
	}
	fmt.Printf("✓ UPanel 已%s\n", actionName(action))
}

func actionName(a string) string {
	switch a {
	case "start":
		return "启动"
	case "stop":
		return "停止"
	default:
		return "重启"
	}
}

func checkStatus() {
	cfg := loadCfg()
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("  安装目录: %s\n", cfg.BaseDir)
	fmt.Printf("  数据目录: %s\n", cfg.DataPath)
	fmt.Printf("  监听地址: %s:%s\n", cfg.BindIP, cfg.Port)
	if entry := cfg.GetSecurityEntry(); entry != "" {
		fmt.Printf("  安全入口: %s\n", entry)
	}
	fmt.Printf("  面板地址: http://%s:%s%s/\n", firstNonLoopback(), cfg.Port, cfg.GetSecurityEntry())

	if !systemctlAvailable() {
		fmt.Println("  服务状态: 未检测到 systemd")
		return
	}
	out, err := exec.Command("systemctl", "is-active", serviceName).Output()
	state := strings.TrimSpace(string(out))
	if err != nil || state != "active" {
		fmt.Printf("  服务状态: ✗ 未运行 (%s)\n", state)
	} else {
		fmt.Println("  服务状态: ✓ 运行中")
	}
	fmt.Println("────────────────────────────────────────")
}

func firstNonLoopback() string {
	// 尽量给出一个可直接访问的 IP，失败则回退占位符
	out, err := exec.Command("sh", "-c",
		"ip -4 addr show scope global 2>/dev/null | grep -oP '(?<=inet\\s)\\d+(\\.\\d+){3}' | head -n1").Output()
	if err == nil {
		if ip := strings.TrimSpace(string(out)); ip != "" {
			return ip
		}
	}
	return "服务器IP"
}

func getUserInfo() {
	cfg := loadCfg()
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("  面板地址: http://%s:%s%s/\n", firstNonLoopback(), cfg.Port, cfg.GetSecurityEntry())
	fmt.Printf("  用户名:   %s\n", cfg.Username)
	fmt.Println("  密码:     安装时设置的密码（忘记可用 up 9 password 重置）")
	fmt.Printf("  端口:     %s\n", cfg.Port)
	fmt.Println("────────────────────────────────────────")
}

func panelInfo() {
	cfg := loadCfg()
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("  版本:     %s\n", Version)
	fmt.Printf("  安装目录: %s\n", cfg.BaseDir)
	fmt.Printf("  配置文件: %s\n", cfg.ConfigFilePath())
	fmt.Printf("  数据目录: %s\n", cfg.DataPath)
	fmt.Printf("  网站目录: %s\n", cfg.WebsitePath)
	fmt.Printf("  数据库目录: %s\n", cfg.DatabasePath)
	fmt.Printf("  应用目录: %s\n", cfg.AppPath)
	fmt.Printf("  前端目录: %s\n", cfg.StaticPath)
	fmt.Printf("  网关镜像: %s\n", cfg.GatewayImage)
	fmt.Printf("  网关端口: %s / %s\n", cfg.GatewayHTTPPort, cfg.GatewayHTTPSPort)
	fmt.Println("────────────────────────────────────────")
}

func changeListenIP() {
	cfg := loadCfg()
	fmt.Println("切换面板监听 IP")
	fmt.Println("  1. 监听所有 IP (0.0.0.0)")
	fmt.Println("  2. 仅监听本地 (127.0.0.1)")
	fmt.Println("  3. 自定义 IP")
	choice := readLine("请选择 [1]: ")
	if choice == "" {
		choice = "1"
	}
	var ip string
	switch choice {
	case "1":
		ip = "0.0.0.0"
	case "2":
		ip = "127.0.0.1"
	case "3":
		ip = readLine("请输入 IP: ")
	default:
		fmt.Println("无效选择")
		return
	}
	if ip == "" {
		fmt.Println("IP 不能为空")
		return
	}
	cfg.BindIP = ip
	if err := cfg.Save(); err != nil {
		fmt.Println("✗ 保存失败:", err)
		return
	}
	fmt.Printf("✓ 监听地址已设置为 %s，面板端口 %s\n", ip, cfg.Port)
	ctl("restart")
}

func showVersion() {
	cfg := loadCfg()
	fmt.Printf("UPanel 版本: %s\n", Version)
	fmt.Printf("面板端口:    %s\n", cfg.Port)
}

func modifySystem(sub, val string) {
	cfg := loadCfg()

	if sub == "" {
		sub = readLine("修改项 (username/password/port/entry): ")
	}
	sub = strings.ToLower(strings.TrimSpace(sub))

	switch sub {
	case "username":
		name := val
		if name == "" {
			name = readLine("请输入新用户名: ")
		}
		name = strings.TrimSpace(name)
		if name == "" || len(name) > 64 || strings.ContainsAny(name, "\r\n\x00") {
			fmt.Println("✗ 用户名不能为空且最多 64 个字符")
			return
		}
		cfg.Username = name
		if err := cfg.Save(); err != nil {
			fmt.Println("✗ 保存失败:", err)
			return
		}
		fmt.Printf("✓ 用户名已修改为: %s\n", name)
		ctl("restart")

	case "password":
		pwd := val
		if pwd == "" {
			pwd = readPassword("请输入新密码: ")
			pwd2 := readPassword("请再次输入新密码: ")
			if pwd != pwd2 {
				fmt.Println("✗ 两次输入的密码不一致")
				return
			}
		}
		if len(pwd) < service.MinPasswordLength {
			fmt.Printf("✗ 密码长度不能少于 %d 位，且需同时包含字母和数字\n", service.MinPasswordLength)
			return
		}
		auth := service.NewAuthService()
		if err := auth.SetPassword(pwd); err != nil {
			fmt.Println("✗ 修改失败:", err)
			return
		}
		fmt.Println("✓ 密码已修改，无需重启，立即生效")

	case "port":
		port := val
		if port == "" {
			port = readLine("请输入新端口号: ")
		}
		port = strings.TrimSpace(port)
		if err := config.ValidatePort(port); err != nil {
			fmt.Println("✗", err)
			return
		}
		cfg.Port = port
		if err := cfg.Save(); err != nil {
			fmt.Println("✗ 保存失败:", err)
			return
		}
		fmt.Printf("✓ 端口已修改为: %s\n", port)
		ctl("restart")

	case "entry":
		entry := val
		if entry == "" {
			entry = readLine("请输入安全入口（如 mypanel，输入 - 取消）: ")
		}
		entry = strings.TrimSpace(entry)
		if entry == "-" {
			entry = ""
		}
		normalized, err := config.NormalizeSecurityEntry(entry)
		if err != nil {
			fmt.Println("✗", err)
			return
		}
		entry = normalized
		cfg.SecurityEntry = strings.TrimPrefix(entry, "/")
		if err := cfg.Save(); err != nil {
			fmt.Println("✗ 保存失败:", err)
			return
		}
		if entry == "" {
			fmt.Println("✓ 已取消安全入口")
		} else {
			fmt.Printf("✓ 安全入口已设置为 %s\n", entry)
		}
		ctl("restart")

	default:
		fmt.Println("可用子命令: username, password, port, entry")
	}
}

func resetSystem() {
	cfg := loadCfg()
	fmt.Println("警告: 此操作将把面板端口重置为 8080、用户名重置为 admin，并生成一个随机密码")
	if !confirm("确认重置？(y/N): ") {
		fmt.Println("已取消")
		return
	}
	cfg.Port = "8080"
	cfg.Username = "admin"
	cfg.SecurityEntry = ""
	if err := cfg.Save(); err != nil {
		fmt.Println("✗ 保存失败:", err)
		return
	}
	password := config.RandomPassword(16)
	auth := service.NewAuthService()
	if err := auth.SetPassword(password); err != nil {
		fmt.Println("✗ 密码重置失败:", err)
		return
	}
	fmt.Println("✓ 已重置：端口 8080 / 用户名 admin")
	fmt.Printf("✓ 新的随机密码: %s\n", password)
	fmt.Println("  请立即登录并修改密码")
	ctl("restart")
}

func restoreData() {
	cfg := loadCfg()
	fmt.Println("此操作用于面板数据目录丢失后恢复：重新创建目录结构并修复权限。")
	if !confirm("确认继续？(y/N): ") {
		fmt.Println("已取消")
		return
	}
	if err := cfg.EnsureDirs(); err != nil {
		fmt.Println("✗ 恢复失败:", err)
		return
	}
	// 重建所有站点/数据库目录的默认结构
	for _, sub := range []string{cfg.WebsitePath, cfg.DatabasePath, cfg.AppPath} {
		_ = os.MkdirAll(sub, 0755)
	}
	fmt.Println("✓ 目录结构已恢复，请重启面板")
	ctl("restart")
}

func gatewayStatus() {
	ws, err := service.NewWebsiteService()
	if err != nil {
		fmt.Println("✗ 无法连接 Docker:", err)
		return
	}
	defer ws.Close()
	state, port := ws.GatewayStatus()
	fmt.Printf("网关容器: upanel-gateway\n")
	fmt.Printf("状态:     %s\n", state)
	fmt.Printf("HTTP 端口: %s\n", port)
	if state != "running" {
		fmt.Println("提示: 可在面板「网站」页点击修复，或执行 systemctl restart upanel 后重试")
	}
}

func showLogs() {
	if !systemctlAvailable() {
		fmt.Println("未检测到 systemd")
		return
	}
	cmd := exec.Command("journalctl", "-u", serviceName, "-n", "100", "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func uninstall() {
	cfg := loadCfg()
	fmt.Println("════════════════════════════════════════")
	fmt.Println("警告: 此操作将停止并移除 UPanel 服务")
	fmt.Printf("安装目录: %s\n", cfg.BaseDir)
	fmt.Println("════════════════════════════════════════")
	keepData := confirm("是否保留网站/数据库等数据目录？(y/N): ")
	if !confirm("确认卸载 UPanel？(y/N): ") {
		fmt.Println("已取消")
		return
	}

	if systemctlAvailable() {
		_ = exec.Command("systemctl", "stop", serviceName).Run()
		_ = exec.Command("systemctl", "disable", serviceName).Run()
	}

	// 停止并清理 UPanel 托管的容器
	if rt, err := service.NewRuntime(); err == nil {
		list, _ := rt.ListManaged()
		for _, name := range list {
			fmt.Println("  移除容器:", name)
			_ = rt.Remove(name)
		}
		rt.Close()
	}

	if !keepData {
		_ = os.RemoveAll(cfg.DataPath)
	}
	_ = os.RemoveAll(filepath.Join(cfg.BaseDir, "bin"))
	_ = os.RemoveAll(filepath.Join(cfg.BaseDir, "web"))
	_ = os.Remove(filepath.Join("/etc/systemd/system", serviceName+".service"))
	if systemctlAvailable() {
		_ = exec.Command("systemctl", "daemon-reload").Run()
	}
	_ = os.Remove("/usr/local/bin/up")

	if keepData {
		fmt.Printf("✓ UPanel 已卸载，数据保留在 %s\n", cfg.DataPath)
	} else {
		fmt.Println("✓ UPanel 已完全卸载")
	}
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	// 关闭回显
	_ = exec.Command("stty", "-echo").Run()
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	_ = exec.Command("stty", "echo").Run()
	fmt.Println()
	return strings.TrimSpace(line)
}

func confirm(prompt string) bool {
	ans := strings.ToLower(readLine(prompt))
	return ans == "y" || ans == "yes"
}
