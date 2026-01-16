package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitStatus Git 状态
type GitStatus struct {
	Branch  string      `json:"branch"`
	Changes []GitChange `json:"changes"`
	HasRepo bool        `json:"hasRepo"`
}

// GitChange Git 变更
type GitChange struct {
	Path   string `json:"path"`
	Status string `json:"status"` // M=modified, A=added, D=deleted, ??=untracked
	Staged bool   `json:"staged"`
}

// GetGitStatus 获取 Git 状态
func (a *App) GetGitStatus(dir string) (*GitStatus, error) {
	if dir == "" {
		return &GitStatus{HasRepo: false}, nil
	}

	// 检查是否是 Git 仓库
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return &GitStatus{HasRepo: false}, nil
	}

	status := &GitStatus{HasRepo: true, Changes: []GitChange{}}

	// 获取当前分支
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		status.Branch = strings.TrimSpace(string(output))
	}

	// 获取状态
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return status, err
	}

	// 解析状态
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		statusCode := line[0:2]
		path := strings.TrimSpace(line[3:])

		change := GitChange{Path: path}

		// 解析状态码
		switch {
		case strings.HasPrefix(statusCode, "M"):
			change.Status = "M"
			change.Staged = statusCode[0] == 'M'
		case strings.HasPrefix(statusCode, "A"):
			change.Status = "A"
			change.Staged = true
		case strings.HasPrefix(statusCode, "D"):
			change.Status = "D"
			change.Staged = statusCode[0] == 'D'
		case strings.HasPrefix(statusCode, "??"):
			change.Status = "??"
			change.Staged = false
		case strings.HasPrefix(statusCode, "R"):
			change.Status = "R"
			change.Staged = true
		default:
			change.Status = "M"
			change.Staged = false
		}

		status.Changes = append(status.Changes, change)
	}

	return status, nil
}

// GitAdd 添加文件到暂存区
func (a *App) GitAdd(dir, path string) error {
	cmd := exec.Command("git", "add", path)
	cmd.Dir = dir
	return cmd.Run()
}

// GitCommit 提交更改
func (a *App) GitCommit(dir, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// GitPush 推送到远程
func (a *App) GitPush(dir string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// GitPull 从远程拉取
func (a *App) GitPull(dir string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// GitDiscard 丢弃更改
func (a *App) GitDiscard(dir, path string) error {
	cmd := exec.Command("git", "checkout", "--", path)
	cmd.Dir = dir
	return cmd.Run()
}
