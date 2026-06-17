package io

import (
	"os"
	"path/filepath"
	"strings"
)

func WriteSnippet(outDir, filename string, lines []string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	tmpFile := filepath.Join(outDir, filename+".tmp")
	finalFile := filepath.Join(outDir, filename)

	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	for _, line := range lines {
		line = AppendStrategyIfNeeded(line, filename)
		if _, err := f.WriteString(line + "\n"); err != nil {
			f.Close()
			os.Remove(tmpFile)
			return err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpFile)
		return err
	}
	return os.Rename(tmpFile, finalFile)
}

func AppendStrategyIfNeeded(line, filename string) string {
	if line == "" || filename == "rewrite.snippet" {
		return line
	}
	strategy := ""
	switch filename {
	case "direct.snippet":
		strategy = "direct"
	case "proxy.snippet":
		strategy = "proxy"
	case "reject.snippet":
		strategy = "reject"
	default:
		return line
	}
	if strings.HasSuffix(line, ","+strategy) {
		return line
	}
	return line + "," + strategy
}