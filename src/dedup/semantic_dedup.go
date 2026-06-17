package dedup

import (
	"strings"

	"github.com/quantumult-x/gen/src/log"
)

func SemanticDedup(lines []string) ([]string, []string, error) {
	index := NewSemanticIndex()
	var kept, removed []string
	for _, line := range lines {
		r := parseRule(line)
		if r == nil {
			kept = append(kept, line)
			continue
		}
		if index.IsCovered(*r) {
			log.Debug("semantic dedup removed: %s", line)
			removed = append(removed, line)
			continue
		}
		kept = append(kept, line)
		index.Add(*r)
	}
	return kept, removed, nil
}

func parseRule(line string) *Rule {
	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return nil
	}
	ruleType := strings.TrimSpace(parts[0])
	ruleValue := strings.TrimSpace(parts[1])
	switch ruleType {
	case "host-suffix", "host-keyword", "host", "host-wildcard",
		"ip-cidr", "ip6-cidr", "ip-asn", "geoip":
		return &Rule{Type: ruleType, Value: ruleValue}
	default:
		return nil
	}
}