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
	pattern  string
	normBody string
	ci       bool
	cat      string
}

func isRedundantEscape(c byte) bool {
	switch c {
	case '/', ':', ';', '!', '#', '&', '=', '@', '%', ',', '~':
		return true
	}
	return false
}

func stripRedundantEscapes(pattern string) string {
	var b strings.Builder
	b.Grow(len(pattern))
	i := 0
	for i < len(pattern) {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			next := pattern[i+1]
			if isRedundantEscape(next) {
				b.WriteByte(next)
				i += 2
				continue
			}
			b.WriteByte('\\')
			b.WriteByte(next)
			i += 2
			continue
		}
		b.WriteByte(pattern[i])
		i++
	}
	return b.String()
}

func normalizePattern(pattern string) (string, bool) {
	ci := false
	p := pattern
	if strings.HasPrefix(p, "(?i)") {
		ci = true
		p = p[4:]
	} else if strings.HasPrefix(p, "(?I)") {
		ci = true
		p = p[4:]
	}
	p = strings.ReplaceAll(p, `\b`, "")
	p = strings.ReplaceAll(p, `\B`, "")
	p = stripRedundantEscapes(p)
	return p, ci
}

func normalizeForOutput(pattern string) string {
	return stripRedundantEscapes(pattern)
}

func guardPasses(normBody string) bool {
	if len(normBody) >= 6 {
		return true
	}
	if strings.Contains(normBody, "/") {
		return true
	}
	if strings.HasPrefix(normBody, "^") {
		return true
	}
	return false
}

func containsCI(hay, needle string, ci bool) bool {
	if ci {
		return strings.Contains(strings.ToLower(hay), strings.ToLower(needle))
	}
	return strings.Contains(hay, needle)
}

func isCoveredByKept(pattern string, ci bool, cat string, keptPatterns []keptPattern) bool {
	newNorm, newCI := normalizePattern(pattern)
	for _, kp := range keptPatterns {
		if kp.normBody == "" {
			continue
		}
		if !isActionCompatible(kp.cat, cat) {
			continue
		}
		if newCI && !kp.ci {
			continue
		}
		if !guardPasses(kp.normBody) {
			continue
		}
		if containsCI(newNorm, kp.normBody, kp.ci) {
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
		if len(kp.normBody) <= len(pattern) && isActionCompatible(kp.cat, cat) && guardPasses(kp.normBody) {
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
			expandedNorm, expandedCI := normalizePattern(expanded)
			covered := false
			for _, kp := range shortPatterns {
				if expandedCI && !kp.ci {
					continue
				}
				if containsCI(expandedNorm, kp.normBody, kp.ci) {
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

		if isCoveredByKept(r.Pattern, patternHasCI(r.Pattern), r.Cat, keptPatterns) {
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

		normalizedOut := normalizeForOutput(r.Pattern)
		if normalizedOut != r.Pattern {
			log.Debug("rewrite dedup normalized escapes: %s -> %s", r.Pattern, normalizedOut)
			line = normalizedOut + line[len(r.Pattern):]
			r.Pattern = normalizedOut
		}

		normBody, ci := normalizePattern(r.Pattern)
		keptPatterns = append(keptPatterns, keptPattern{
			pattern:  r.Pattern,
			normBody: normBody,
			ci:       ci,
			cat:      r.Cat,
		})

		if accelDomain != "" {
			line = replaceGithubURLs(line, accelDomain)
		}
		kept = append(kept, line)
	}
	return kept, removed
}

func patternHasCI(pattern string) bool {
	return strings.HasPrefix(pattern, "(?i)") || strings.HasPrefix(pattern, "(?I)")
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
