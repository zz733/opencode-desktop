package main

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"
)

// SearchResult 搜索结果
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Match   string `json:"match"`
}

// SearchInFiles 在文件中搜索
func (a *App) SearchInFiles(dir, query string, caseSensitive, useRegex bool) ([]SearchResult, error) {
	if dir == "" || query == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult
	var cmd *exec.Cmd

	if useRegex {
		// 使用 grep 进行正则搜索
		args := []string{"-rn", "-E"}
		if !caseSensitive {
			args = append(args, "-i")
		}
		args = append(args, query, dir)
		
		if goruntime.GOOS == "windows" {
			cmd = exec.Command("findstr", "/S", "/N", query, dir+"\\*")
		} else {
			cmd = exec.Command("grep", args...)
		}
	} else {
		// 简单文本搜索
		args := []string{"-rn", "-F"}
		if !caseSensitive {
			args = append(args, "-i")
		}
		args = append(args, query, dir)
		
		if goruntime.GOOS == "windows" {
			cmd = exec.Command("findstr", "/S", "/N", query, dir+"\\*")
		} else {
			cmd = exec.Command("grep", args...)
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep 没找到结果会返回错误，这是正常的
		if len(output) == 0 {
			return []SearchResult{}, nil
		}
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// 格式: path:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		lineNum := 0
		fmt.Sscanf(parts[1], "%d", &lineNum)

		results = append(results, SearchResult{
			Path:    parts[0],
			Line:    lineNum,
			Content: strings.TrimSpace(parts[2]),
			Match:   query,
		})

		// 限制结果数量
		if len(results) >= 500 {
			break
		}
	}

	return results, nil
}
