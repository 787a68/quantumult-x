package rewrite

import (
	"regexp"
	"strings"

	"github.com/quantumult-x/gen/src/log"
)

var githubRe = regexp.MustCompile(`https://(github\.com|raw\.github(?:usercontent)?\.com)(/[^\s"]+)`)

var branchGroupRe = regexp.MustCompile(`\(([^()|]+(?:\|[^()|]+)+)\)`)

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

func isCoveredByKept(pattern string, cat string, keptPatterns []keptPattern) bool {
	for _, kp := range keptPatterns {
		if kp.pattern != "" && isActionCompatible(kp.cat, cat) && strings.Contains(pattern, kp.pattern) {
			return true
		}
	}
	return false
}

func tryBranchExpansion(pattern string, cat string, keptPatterns []keptPattern) (newPattern string, fullyCovered bool) {
	if !strings.Contains(pattern, "(") {
		return pattern, false
	}

	var shortPatterns []keptPattern
	for _, kp := range keptPatterns {
		if len(kp.pattern) <= len(pattern) && isActionCompatible(kp.cat, cat) {
			shortPatterns = append(shortPatterns, kp)
		}
	}
	if len(shortPatterns) == 0 {
		return pattern, false
	}

	newPattern = pattern
	searchFrom := 0
	for {
		loc := branchGroupRe.FindStringSubmatchIndex(newPattern[searchFrom:])
		if loc == nil {
			return newPattern, false
		}
		absStart := searchFrom + loc[0]
		absEnd := searchFrom + loc[1]
		groupContent := newPattern[searchFrom+loc[2] : searchFrom+loc[3]]
		branches := strings.Split(groupContent, "|")

		var remaining []string
		for _, branch := range branches {
			expanded := newPattern[:absStart] + branch + newPattern[absEnd:]
			covered := false
			for _, kp := range shortPatterns {
				if kp.pattern != "" && strings.Contains(expanded, kp.pattern) {
					covered = true
					break
				}
			}
			if !covered {
				remaining = append(remaining, branch)
			}
		}

		if len(remaining) == 0 {
			return "", true
		}

		var replacement string
		if len(remaining) == 1 {
			replacement = remaining[0]
		} else {
			replacement = "(" + strings.Join(remaining, "|") + ")"
		}

		changed := replacement != newPattern[absStart:absEnd]
		newPattern = newPattern[:absStart] + replacement + newPattern[absEnd:]
		searchFrom = absStart + len(replacement)

		if !changed {
			searchFrom = absEnd
		}
	}
}

func RewriteSemanticDedup(lines []string, accelDomain string) ([]string, []string) {
	var kept, removed []string
	var keptPatterns []keptPattern

	for _, line := range lines {
		r := parseRewriteLine(line)

		if isCoveredByKept(r.Pattern, r.Cat, keptPatterns) {
			log.Debug("rewrite dedup removed: %s", line)
			removed = append(removed, line)
			continue
		}

		newPattern, fullyCovered := tryBranchExpansion(r.Pattern, r.Cat, keptPatterns)
		if fullyCovered {
			log.Debug("rewrite dedup removed (branch fully covered): %s", line)
			removed = append(removed, line)
			continue
		}

		if newPattern != r.Pattern {
			log.Debug("rewrite dedup branch pruned: %s -> %s", r.Pattern, newPattern)
			line = newPattern + line[len(r.Pattern):]
			r.Pattern = newPattern
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
