package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type Heading struct {
	File  string
	Line  int
	Level int
	Text  string
	Tags  []string
}

func main() {
	allFlag := flag.Bool("all", false, "List all headings")
	rootDir := flag.String("path", ".", "Search directory")
	flag.Parse()

	var keyword string
	if flag.NArg() > 0 {
		keyword = strings.ToLower(flag.Arg(0))
	}

	files, err := scanMarkdownFiles(*rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan error: %v\n", err)
		os.Exit(1)
	}

	var allHeadings []Heading
	for _, f := range files {
		hs, err := parseHeadings(f)
		if err == nil {
			allHeadings = append(allHeadings, hs...)
		}
	}

	var result []Heading
	if *allFlag {
		result = allHeadings
	} else if keyword != "" {
		for _, h := range allHeadings {
			textLower := strings.ToLower(h.Text)
			if strings.Contains(textLower, keyword) {
				result = append(result, h)
				continue
			}
			// 如果关键字在 tag 中也匹配
			matchedInTag := false
			for _, t := range h.Tags {
				if strings.Contains(strings.ToLower(t), keyword) {
					matchedInTag = true
					break
				}
			}
			if matchedInTag {
				result = append(result, h)
			}
		}
	} else {
		fmt.Println("请提供搜索关键词，或使用 --all")
		return
	}

	// 排序规则：
	// 1) 带 @tag 的优先（无论 tag 在哪儿）
	// 2) level (# 数量) 升序：# 少的优先
	// 3) 文件名 + 行号作为 tiebreaker
	sort.Slice(result, func(i, j int) bool {
		iHasTag := len(result[i].Tags) > 0
		jHasTag := len(result[j].Tags) > 0
		if iHasTag != jHasTag {
			return iHasTag // 有 tag 的在前
		}
		if result[i].Level != result[j].Level {
			return result[i].Level < result[j].Level // # 少的在前
		}
		if result[i].File != result[j].File {
			return result[i].File < result[j].File
		}
		return result[i].Line < result[j].Line
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"File", "Line", "Level", "Title", "Tags"})
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Header.Formatting.AutoWrap = 0
	})

	for _, h := range result {
		// level 字段用 # 表示并着色
		levelStr := strings.Repeat("#", h.Level)
		levelStr = color.New(color.FgYellow).Sprint(levelStr)

		// 高亮 tags（在标题旁边也列出彩色 tags）
		titleStr := h.Text
		if len(h.Tags) > 0 {
			tagColored := []string{}
			for _, t := range h.Tags {
				tagColored = append(tagColored, color.New(color.FgHiGreen).Sprint("@"+t))
			}
			// 把彩色 tags 附到标题后便于观察
			titleStr = fmt.Sprintf("%s  %s", h.Text, strings.Join(tagColored, " "))
		}

		table.Append([]string{
			h.File,
			fmt.Sprintf("%d", h.Line),
			levelStr,
			titleStr,
			strings.Join(h.Tags, ", "),
		})
	}

	table.Render()

	// 如果不是 --all 并且结果不为空 → 进入 fzf
	if !*allFlag && len(result) > 0 {
		chosen, err := pickWithFzf(result)
		if err == nil {
			openFileAtLine(chosen.File, chosen.Line)
			return
		}
	}

}

// scanMarkdownFiles recursively finds all .md files under root
func scanMarkdownFiles(root string) ([]string, error) {
	var results []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 忽略某些无法进入的目录，而不是直接返回 err
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			results = append(results, path)
		}
		return nil
	})
	return results, err
}

// parseHeadings extracts headings from a markdown file
// 匹配类似：   ### @Tag something ...   （前面允许空格）
func parseHeadings(path string) ([]Heading, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []Heading
	// 允许前导空格，捕获 # 数量，后面取整行作为标题文本
	re := regexp.MustCompile(`^\s*(#+)\s+(.*\S)\s*$`)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		match := re.FindStringSubmatch(line)
		if len(match) == 3 {
			level := len(match[1])
			text := strings.TrimSpace(match[2])

			var tags []string
			for _, f := range strings.Fields(text) {
				if strings.HasPrefix(f, "@") && len(f) > 1 {
					// 去掉前导 @，保留 tag 内容
					tags = append(tags, strings.TrimPrefix(f, "@"))
				}
			}

			results = append(results, Heading{
				File:  path,
				Line:  lineNum,
				Level: level,
				Text:  text,
				Tags:  tags,
			})
		}
	}
	return results, scanner.Err()
}
