package main

import (
"os"
"strings"
)

func getOpenDirectoryText() string {
	lang := os.Getenv("LANG")
	if strings.HasPrefix(lang, "zh") {
		return "打开目录"
	} else if strings.HasPrefix(lang, "ja") {
		return "フォルダを開く"
	}
	return "Open Directory"
}
