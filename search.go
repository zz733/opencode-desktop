package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// ReplaceInFiles 在文件中替换文本
func (a *App) ReplaceInFiles(dir, searchText, replaceText string, caseSensitive bool) (int, error) {
	if dir == "" || searchText == "" {
		return 0, fmt.Errorf("搜索文本不能为空")
	}

	count := 0
	
	// 遍历目录下所有文件
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误的文件
		}

		// 跳过目录和隐藏文件/目录
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过二进制文件和大文件
		if info.Size() > 10*1024*1024 { // 10MB
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// 检查是否是文本文件（简单检测）
		if !isTextFile(content) {
			return nil
		}

		contentStr := string(content)
		var newContent string
		var replaced bool

		if caseSensitive {
			if strings.Contains(contentStr, searchText) {
				newContent = strings.ReplaceAll(contentStr, searchText, replaceText)
				replaced = true
			}
		} else {
			// 大小写不敏感替换
			lower := strings.ToLower(contentStr)
			searchLower := strings.ToLower(searchText)
			if strings.Contains(lower, searchLower) {
				newContent = replaceInsensitive(contentStr, searchText, replaceText)
				replaced = true
			}
		}

		if replaced {
			if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
				return nil
			}
			count++
		}

		return nil
	})

	return count, err
}

// isTextFile 简单判断是否是文本文件
func isTextFile(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	
	// 检查前 512 字节
	sample := content
	if len(content) > 512 {
		sample = content[:512]
	}
	
	// 如果包含太多非打印字符，认为是二进制文件
	nonPrintable := 0
	for _, b := range sample {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}
	
	return float64(nonPrintable)/float64(len(sample)) < 0.3
}

// replaceInsensitive 大小写不敏感替换
func replaceInsensitive(s, old, new string) string {
	if old == "" {
		return s
	}
	
	oldLower := strings.ToLower(old)
	result := ""
	remaining := s
	
	for {
		remainingLower := strings.ToLower(remaining)
		idx := strings.Index(remainingLower, oldLower)
		if idx == -1 {
			result += remaining
			break
		}
		
		result += remaining[:idx] + new
		remaining = remaining[idx+len(old):]
	}
	
	return result
}

