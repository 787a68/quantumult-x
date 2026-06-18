package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/quantumult-x/gen/src/clean"
	"github.com/quantumult-x/gen/src/config"
	"github.com/quantumult-x/gen/src/dedup"
	"github.com/quantumult-x/gen/src/fetch"
	ioW "github.com/quantumult-x/gen/src/io"
	"github.com/quantumult-x/gen/src/log"
	"github.com/quantumult-x/gen/src/rewrite"
	"github.com/quantumult-x/gen/src/transform"
	"github.com/quantumult-x/gen/src/util"
)

type confInfo struct {
	filename  string
	isRewrite bool
}

var confFiles = []confInfo{
	{"direct.conf", false},
	{"proxy.conf", false},
	{"reject.conf", false},
	{"rewrite.conf", true},
}

func main() {
	confDir := flag.String("conf-dir", ".", "directory containing .conf files")
	outDir := flag.String("out-dir", "out", "output directory")
	configPath := flag.String("config", "src/config/config.yaml", "path to config.yaml")
	dryRun := flag.Bool("dry-run", false, "dry run mode, skip publish")
	useExamples := flag.Bool("examples", false, "use examples/ as upstream source (local files)")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	log.SetLevel(cfg.LogLevel)

	if err := runAll(*confDir, *outDir, cfg, *dryRun, *useExamples); err != nil {
		log.Error("fatal: %v", err)
		os.Exit(1)
	}
}

func runAll(confDir, outDir string, cfg *config.Config, dryRun, useExamples bool) error {
	accelDomain := os.Getenv("ACCEL_DOMAIN")

	for _, ci := range confFiles {
		confPath := filepath.Join(confDir, ci.filename)
		snippetName := strings.TrimSuffix(ci.filename, ".conf") + ".snippet"

		log.Info("processing %s -> %s", ci.filename, snippetName)

		spec, err := clean.ParseConf(confPath)
		if err != nil {
			return fmt.Errorf("parse conf %s: %w", ci.filename, err)
		}

		var allLines []string
		if useExamples {
			allLines, err = processLocalSources(spec, confDir, ci.isRewrite)
		} else {
			allLines, err = processRemoteSources(spec, cfg, ci.isRewrite)
		}
		if err != nil {
			return fmt.Errorf("process sources for %s: %w", ci.filename, err)
		}

		if ci.isRewrite {
			if err := processRewrite(allLines, spec, outDir, snippetName, accelDomain); err != nil {
				return err
			}
			continue
		}

		deduped, textRemoved := dedup.TextDedup(allLines)
		log.Info("text dedup: %d -> %d (removed %d)", len(allLines), len(deduped), textRemoved)

		deduped = sortRules(deduped)
		deduped = insertHeadRules(deduped, spec.HeadRules)

		kept, removed, err := dedup.SemanticDedup(deduped)
		if err != nil {
			return fmt.Errorf("semantic dedup for %s: %w", ci.filename, err)
		}
		log.Info("semantic dedup: %d -> %d (removed %d)", len(deduped), len(kept), len(removed))

		kept = applyExcludes(kept, spec.Excludes)

		log.Info("writing %d lines to %s", len(kept), snippetName)
		if err := ioW.WriteSnippet(outDir, snippetName, kept); err != nil {
			return fmt.Errorf("write %s: %w", snippetName, err)
		}
	}
	return nil
}

func processRewrite(lines []string, spec *clean.ConfSpec, outDir, snippetName, accelDomain string) error {
	var rewriteLines []string
	var hostnameEntries []string
	for _, line := range lines {
		if strings.HasPrefix(line, "hostname") && strings.Contains(line, "=") {
			fields := strings.SplitN(line, "=", 2)
			if len(fields) == 2 {
				hosts := strings.Split(fields[1], ",")
				for _, h := range hosts {
					hostnameEntries = append(hostnameEntries, strings.TrimSpace(h))
				}
			}
		} else {
			rewriteLines = append(rewriteLines, line)
		}
	}
	log.Info("rewrite: %d rules, %d hostname entries", len(rewriteLines), len(hostnameEntries))

	sources := [][]string{rewriteLines}
	sorted, err := rewrite.MergeRewrite(sources)
	if err != nil {
		return fmt.Errorf("merge rewrite: %w", err)
	}
	log.Info("rewrite sort: %d lines", len(sorted))

	sorted = insertHeadRules(sorted, spec.HeadRules)

	deduped, textRemoved := dedup.TextDedup(sorted)
	log.Info("rewrite text dedup: %d -> %d (removed %d)", len(sorted), len(deduped), textRemoved)

	kept, removed := rewrite.RewriteSemanticDedup(deduped, accelDomain)
	log.Info("rewrite semantic dedup: %d -> %d (removed %d)", len(deduped), len(kept), len(removed))

	kept = applyExcludes(kept, spec.Excludes)

	if len(hostnameEntries) > 0 {
		uniqueHosts := dedupText(hostnameEntries)
		kept = append(kept, "", "hostname = "+strings.Join(uniqueHosts, ","))
	}

	log.Info("writing %d lines to %s", len(kept), snippetName)
	return ioW.WriteSnippet(outDir, snippetName, kept)
}

func dedupText(items []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func processLocalSources(spec *clean.ConfSpec, confDir string, isRewrite bool) ([]string, error) {
	var allLines []string
	examplesDir := filepath.Join(confDir, "examples")
	for _, src := range spec.Upstreams {
		base := filepath.Base(src.URL)
		candidates := []string{
			filepath.Join(examplesDir, base),
			filepath.Join(examplesDir, base+".txt"),
		}
		var f *os.File
		var opened bool
		for _, p := range candidates {
			if fp, err := os.Open(p); err == nil {
				f = fp
				opened = true
				break
			}
		}
		if !opened {
			log.Warn("local source not found: %s (tried %v)", base, candidates)
			continue
		}
		lines := readAndTransform(f, src.Format, isRewrite)
		f.Close()
		allLines = append(allLines, lines...)
	}
	return allLines, nil
}

func processRemoteSources(spec *clean.ConfSpec, cfg *config.Config, isRewrite bool) ([]string, error) {
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
		transformed, err := transformLine(line, format)
		if err != nil {
			log.Warn("transform failed for %q (format=%s): %v", line, format, err)
			continue
		}
		lines = append(lines, transformed)
	}
	return lines
}

func transformLine(line, format string) (string, error) {
	switch format {
	case "rule":
		result, err := transform.TransformRuleLine(line)
		if err == nil {
			return result, nil
		}
		result, err2 := transform.TransformQxLine(line)
		if err2 == nil {
			return result, nil
		}
		return "", err
	case "set":
		return transform.TransformSetLine(line)
	case "qx", "":
		result, err := transform.TransformQxLine(line)
		if err == nil {
			return result, nil
		}
		result, err2 := transform.TransformRuleLine(line)
		if err2 == nil {
			return result, nil
		}
		return "", err
	default:
		return transform.TransformQxLine(line)
	}
}

func sortRules(lines []string) []string {
	typeGroups := map[int][]string{}
	order := func(line string) int {
		parts := strings.SplitN(line, ",", 2)
		switch parts[0] {
		case "host-keyword":
			return 0
		case "host-wildcard":
			return 1
		case "host-suffix":
			return 2
		case "host":
			return 3
		case "ip-cidr":
			return 4
		case "ip6-cidr":
			return 5
		case "ip-asn":
			return 6
		case "geoip":
			return 7
		default:
			return 8
		}
	}

	var keys []int
	for _, line := range lines {
		o := order(line)
		if _, exists := typeGroups[o]; !exists {
			keys = append(keys, o)
		}
		typeGroups[o] = append(typeGroups[o], line)
	}
	sort.Ints(keys)

	var result []string
	for _, k := range keys {
		group := typeGroups[k]
		sort.Strings(group)
		result = append(result, group...)
	}
	return result
}

func insertHeadRules(lines, headRules []string) []string {
	if len(headRules) == 0 {
		return lines
	}
	return append(headRules, lines...)
}

func applyExcludes(lines []string, excludes []string) []string {
	if len(excludes) == 0 {
		return lines
	}
	var result []string
	for _, line := range lines {
		excluded := false
		for _, exc := range excludes {
			if strings.HasPrefix(line, exc) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, line)
		}
	}
	return result
}