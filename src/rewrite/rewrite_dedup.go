package rewrite

import (
	"regexp"
	"strings"

	"github.com/quantumult-x/gen/src/log"
)

var githubRe = regexp.MustCompile(`https://(github\.com|raw\.github(?:usercontent)?\.com)(/[^\s"]+)`)

func isRejectFamily(cat string) bool {
	switch cat {
	case "reject", "reject-200", "reject-img", "reject-dict", "reject-array":
		return true
	default:
		return false
	}
}

func isActionCompatible(existingCat, newCat string) bool {
	if isRejectFamily(existingCat) {
		return true
	}
	return existingCat == newCat
}

type keptPattern struct {
	pattern string
	cat     string
}

func RewriteSemanticDedup(lines []string, accelDomain string) ([]string, []string) {
	var kept, removed []string
	var keptPatterns []keptPattern

	for _, line := range lines {
		r := parseRewriteLine(line)

		covered := false
		for _, kp := range keptPatterns {
			if kp.pattern != "" && isActionCompatible(kp.cat, r.Cat) && strings.Contains(r.Pattern, kp.pattern) {
				log.Debug("rewrite dedup removed: %s (covered by pattern %s)", line, kp.pattern)
				covered = true
				break
			}
		}

		if covered {
			removed = append(removed, line)
			continue
		}

		keptPatterns = append(keptPatterns, keptPattern{pattern: r.Pattern, cat: r.Cat})

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