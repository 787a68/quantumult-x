package rewrite

import (
	"strings"

	"github.com/quantumult-x/gen/src/clean"
	"github.com/quantumult-x/gen/src/dedup"
	"github.com/quantumult-x/gen/src/log"
)

func Process(lines []string, spec *clean.ConfSpec, accelDomain string) ([]string, error) {
	var rewriteLines, hostnameEntries []string
	for _, line := range lines {
		if strings.HasPrefix(line, "hostname") && strings.Contains(line, "=") {
			fields := strings.SplitN(line, "=", 2)
			if len(fields) == 2 {
				for _, h := range strings.Split(fields[1], ",") {
					hostnameEntries = append(hostnameEntries, strings.TrimSpace(h))
				}
			}
		} else {
			rewriteLines = append(rewriteLines, line)
		}
	}
	log.Info("rewrite: %d rules, %d hostname entries", len(rewriteLines), len(hostnameEntries))

	sorted, err := MergeRewrite([][]string{rewriteLines})
	if err != nil {
		return nil, err
	}
	log.Info("rewrite sort: %d lines", len(sorted))

	sorted = spec.InsertHeadRules(sorted)

	deduped, textRemoved := dedup.TextDedup(sorted)
	log.Info("rewrite text dedup: %d -> %d (removed %d)", len(sorted), len(deduped), textRemoved)

	kept, removed := RewriteSemanticDedup(deduped, accelDomain)
	log.Info("rewrite semantic dedup: %d -> %d (removed %d)", len(deduped), len(kept), len(removed))

	kept = spec.ApplyExcludes(kept)

	if len(hostnameEntries) > 0 {
		uniqueHosts := dedup.DedupSorted(hostnameEntries)
		kept = append(kept, "", "hostname = "+strings.Join(uniqueHosts, ","))
	}

	log.Info("writing %d lines", len(kept))
	return kept, nil
}
