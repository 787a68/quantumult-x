package rewrite

import (
	"regexp"
	"strings"

	"github.com/quantumult-x/gen/src/log"
)

var githubRe = regexp.MustCompile(`https://(github\.com|raw\.github(?:usercontent)?\.com)(/[^\s"]+)`)

func RewriteSemanticDedup(lines []string, accelDomain string) ([]string, []string) {
	patternSeen := make(map[string]struct{})
	var kept, removed []string
	for _, line := range lines {
		r := parseRewriteLine(line)
		key := r.Cat + "|" + r.Pattern
		if _, exists := patternSeen[key]; exists {
			log.Debug("rewrite dedup removed: %s", line)
			removed = append(removed, line)
			continue
		}
		patternSeen[key] = struct{}{}

		if accelDomain != "" {
			line = replaceGithubURLs(line, accelDomain)
		}
		kept = append(kept, line)
	}
	return kept, removed
}

func replaceGithubURLs(line, accelDomain string) string {
	accel := strings.TrimRight(accelDomain, "/")
	result := githubRe.ReplaceAllStringFunc(line, func(match string) string {
		sub := githubRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		host := sub[1]
		path := sub[2]
		return "https://" + accel + "/" + host + path
	})
	return result
}