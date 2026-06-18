package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/quantumult-x/gen/src/clean"
	"github.com/quantumult-x/gen/src/config"
	"github.com/quantumult-x/gen/src/dedup"
	ioW "github.com/quantumult-x/gen/src/io"
	"github.com/quantumult-x/gen/src/log"
	"github.com/quantumult-x/gen/src/pipeline"
	"github.com/quantumult-x/gen/src/rewrite"
	"github.com/quantumult-x/gen/src/rules"
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
	useExamples := flag.Bool("examples", false, "use examples/ as upstream source (local files)")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	log.SetLevel(cfg.LogLevel)

	if err := runAll(*confDir, *outDir, cfg, *useExamples); err != nil {
		log.Error("fatal: %v", err)
		os.Exit(1)
	}
}

func runAll(confDir, outDir string, cfg *config.Config, useExamples bool) error {
	accelDomain := os.Getenv("ACCEL_DOMAIN")

	for _, ci := range confFiles {
		confPath := filepath.Join(confDir, ci.filename)
		snippetName := strings.TrimSuffix(ci.filename, ".conf") + ".snippet"

		log.Info("processing %s -> %s", ci.filename, snippetName)

		spec, err := clean.ParseConf(confPath)
		if err != nil {
			return fmt.Errorf("parse conf %s: %w", ci.filename, err)
		}

		lines, err := pipeline.FetchLines(spec, confDir, cfg, useExamples, ci.isRewrite)
		if err != nil {
			return fmt.Errorf("process sources for %s: %w", ci.filename, err)
		}

		var kept []string
		if ci.isRewrite {
			kept, err = rewrite.Process(lines, spec, accelDomain)
			if err != nil {
				return fmt.Errorf("process rewrite for %s: %w", ci.filename, err)
			}
		} else {
			kept = buildRules(lines, spec)
		}

		if err := ioW.WriteSnippet(outDir, snippetName, kept); err != nil {
			return fmt.Errorf("write %s: %w", snippetName, err)
		}
	}
	return nil
}

func buildRules(lines []string, spec *clean.ConfSpec) []string {
	deduped, textRemoved := dedup.TextDedup(lines)
	log.Info("text dedup: %d -> %d (removed %d)", len(lines), len(deduped), textRemoved)

	deduped = rules.Sort(deduped)
	deduped = spec.InsertHeadRules(deduped)

	kept, removed := dedup.SemanticDedup(deduped)
	log.Info("semantic dedup: %d -> %d (removed %d)", len(deduped), len(kept), len(removed))

	kept = spec.ApplyExcludes(kept)

	log.Info("writing %d lines", len(kept))
	return kept
}
