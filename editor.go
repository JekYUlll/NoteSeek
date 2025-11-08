package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func defaultEditor() string {
	if ed := os.Getenv("VISUAL"); ed != "" {
		return ed
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed
	}
	return "xdg-open" // fallback：交给桌面系统
}

// 打开文件并跳到指定行
func openFileAtLine(file string, line int) error {
	switch runtime.GOOS {
	case "windows":
		// Windows 默认跳行不保证支持，所以直接交给默认编辑器
		return exec.Command("cmd", "/c", "start", "", file).Start()

	case "darwin":
		if ed := os.Getenv("EDITOR"); ed != "" {
			return exec.Command(ed, fmt.Sprintf("+%d", line), file).Start()
		}
		return exec.Command("open", file).Start()

	default: // Linux
		if ed := os.Getenv("EDITOR"); ed != "" {
			return exec.Command(ed, fmt.Sprintf("+%d", line), file).Start()
		}
		return exec.Command("xdg-open", file).Start()
	}
}

func pickWithFzf(items []Heading) (*Heading, error) {
	// 构建传给 fzf 的一行一条数据
	var lines []string
	for _, h := range items {
		lines = append(lines, fmt.Sprintf("%s:%d  %s", h.File, h.Line, h.Text))
	}

	cmd := exec.Command("fzf", "--ansi", "--reverse", "--border")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// 找回被选中的 heading
	selected := strings.TrimSpace(string(out))
	for _, h := range items {
		if strings.Contains(selected, fmt.Sprintf("%s:%d", h.File, h.Line)) {
			return &h, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
