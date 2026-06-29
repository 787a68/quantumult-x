package io

import (
	"os"
	"path/filepath"
	"strings"
)

var defaultPolicies = []string{"direct", "proxy", "reject"}

func WriteSnippet(outDir, filename string, lines []string, policies map[string]string) error {
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
		line = AppendStrategyIfNeeded(line, filename, policies)
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

func AppendStrategyIfNeeded(line, filename string, policies map[string]string) string {
	if line == "" {
		return line
	}
	prefix := strings.TrimSuffix(filename, ".snippet")
	strategy, ok := policies[prefix]
	if !ok || strategy == "" {
		return line
	}
	if strings.HasSuffix(line, ","+strategy) {
		return line
	}
	for _, dp := range defaultPolicies {
		if strings.HasSuffix(line, ","+dp) {
			line = strings.TrimSuffix(line, ","+dp)
			break
		}
	}
	return line + "," + strategy
}
