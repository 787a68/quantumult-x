package rewrite

import (
	"regexp"
	"sort"
	"strings"
)

type RewriteRule struct {
	Line    string
	Pattern string
	Cat     string
	SubType string
}

func catOrder(cat string) int {
	switch cat {
	case "reject":
		return 0
	case "reject-img":
		return 1
	case "reject-200":
		return 2
	case "reject-dict":
		return 3
	case "reject-array":
		return 4
	case "url302":
		return 5
	case "url307":
		return 6
	case "request-header":
		return 7
	case "request-body":
		return 8
	case "echo-response":
		return 9
	case "jsonjq-response-body":
		return 10
	case "script-request-header":
		return 11
	case "script-request-body":
		return 12
	case "script-response-header":
		return 13
	case "script-response-body":
		return 14
	case "script-analyze-echo-response":
		return 15
	case "response-header":
		return 16
	case "response-body":
		return 17
	default:
		return 20
	}
}

var urlPrefixRe = regexp.MustCompile(`^(?i)(https?://)`)

func parseRewriteURLPattern(line string) (pattern string, rest string, found bool) {
	loc := urlPrefixRe.FindStringIndex(line)
	if loc == nil {
		m := strings.SplitN(line, " ", 2)
		if len(m) < 2 {
			return line, "", false
		}
		return m[0], m[1], true
	}
	urlEnd := loc[1]
	restStr := line[urlEnd:]
	for i, c := range restStr {
		if c == ' ' {
			pattern = line[: urlEnd + i]
			restStr = restStr[i + 1:]
			return pattern, restStr, true
		}
	}
	return line, "", true
}

func parseRewriteLine(line string) RewriteRule {
	r := RewriteRule{Line: line}

	if strings.Contains(line, " url ") || strings.Contains(line, " url-and-header ") {
		sep := " url "
		if strings.Contains(line, " url-and-header ") {
			sep = " url-and-header "
		}
		parts := strings.SplitN(line, sep, 2)
		if len(parts) < 2 {
			r.Cat = "unknown"
			r.Pattern = line
			return r
		}
		r.Pattern = parts[0]
		actionPart := parts[1]

		actionFields := strings.SplitN(actionPart, " ", 2)
		action := actionFields[0]
		r.Cat = action

		if len(actionFields) > 1 {
			r.SubType = actionFields[1]
		}
		return r
	}

	fields := strings.Fields(line)
	if len(fields) >= 3 && strings.HasPrefix(fields[0], "^") {
		r.Pattern = fields[0]
		if strings.HasPrefix(fields[1], "url") {
			r.Cat = fields[1]
			if len(fields) > 2 {
				r.SubType = strings.Join(fields[2:], " ")
			}
		} else {
			r.Cat = fields[1]
			if len(fields) > 2 {
				r.SubType = strings.Join(fields[2:], " ")
			}
		}
		return r
	}

	r.Pattern = line
	r.Cat = "unknown"
	return r
}

func MergeRewrite(sources [][]string) ([]string, error) {
	var all []RewriteRule
	for _, src := range sources {
		for _, line := range src {
			all = append(all, parseRewriteLine(line))
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		oi := catOrder(all[i].Cat)
		oj := catOrder(all[j].Cat)
		if oi != oj {
			return oi < oj
		}
		return all[i].Line < all[j].Line
	})

	result := make([]string, len(all))
	for i, r := range all {
		result[i] = r.Line
	}
	return result, nil
}