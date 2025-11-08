package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	// If an editor is explicitly set, we try to jump line
	if editor != "" {
		base := filepath.Base(editor)
		switch base {
		case "nvim", "vim":
			return exec.Command(editor, fmt.Sprintf("+%d", line), file).Start()
		case "hx":
			return exec.Command(editor, fmt.Sprintf("%s:%d", file, line)).Start()
		case "code":
			return exec.Command("code", "-g", fmt.Sprintf("%s:%d", file, line)).Start()
		case "subl":
			return exec.Command("subl", fmt.Sprintf("%s:%d", file, line)).Start()
		}
		// Fallback: generic +line
		return exec.Command(editor, fmt.Sprintf("+%d", line), file).Start()
	}

	// No editor set: use platform default
	switch runtime.GOOS {
	case "windows":
		// Windows 没法跳行，只能打开文件
		return exec.Command("cmd", "/c", "start", "", file).Start()
	case "darwin":
		return exec.Command("open", file).Start()
	default:
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
