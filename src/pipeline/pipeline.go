package pipeline

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/quantumult-x/gen/src/clean"
	"github.com/quantumult-x/gen/src/config"
	"github.com/quantumult-x/gen/src/fetch"
	"github.com/quantumult-x/gen/src/log"
	"github.com/quantumult-x/gen/src/transform"
	"github.com/quantumult-x/gen/src/util"
)

func FetchLines(spec *clean.ConfSpec, confDir string, cfg *config.Config, useExamples, isRewrite bool) ([]string, error) {
	if useExamples {
		return processLocal(spec, confDir, isRewrite)
	}
	return processRemote(spec, cfg, isRewrite)
}

func processLocal(spec *clean.ConfSpec, confDir string, isRewrite bool) ([]string, error) {
	var allLines []string
	examplesDir := filepath.Join(confDir, "examples")
	for _, src := range spec.Upstreams {
		base := filepath.Base(src.URL)
		candidates := []string{
			filepath.Join(examplesDir, base),
			filepath.Join(examplesDir, base+".txt"),
		}
		var f *os.File
		for _, p := range candidates {
			if fp, err := os.Open(p); err == nil {
				f = fp
				break
			}
		}
		if f == nil {
			log.Warn("local source not found: %s (tried %v)", base, candidates)
			continue
		}
		lines := readAndTransform(f, src.Format, isRewrite)
		f.Close()
		allLines = append(allLines, lines...)
	}
	return allLines, nil
}

func processRemote(spec *clean.ConfSpec, cfg *config.Config, isRewrite bool) ([]string, error) {
	timeout := time.Duration(cfg.HTTPTimeoutSec) * time.Second
	sources := make([]util.UpstreamSource, len(spec.Upstreams))
	copy(sources, spec.Upstreams)

	results, errors := fetch.FetchAll(sources, cfg.Concurrency, timeout, cfg.HTTPRetries)
	for url, err := range errors {
		log.Warn("fetch error for %s: %v", url, err)
	}

	var allLines []string
	for _, src := range spec.Upstreams {
		reader, ok := results[src.URL]
		if !ok {
			continue
		}
		lines := readAndTransform(reader, src.Format, isRewrite)
		reader.Close()
		allLines = append(allLines, lines...)
	}
	return allLines, nil
}

func readAndTransform(r io.Reader, format string, isRewrite bool) []string {
	var lines []string
	for line := range clean.CleanLines(r) {
		if isRewrite {
			lines = append(lines, line)
			continue
		}
		transformed, err := transform.TransformLine(line, format)
		if err != nil {
			log.Warn("transform failed for %q (format=%s): %v", line, format, err)
			continue
		}
		lines = append(lines, transformed)
	}
	return lines
}
