package rewrite

import (
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
	// Group 0: reject
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

	// Group 1: redirect
	case "302":
		return 100
	case "307":
		return 101

	// Group 2: request (sub: header=0, echo=1, analyze=2, body=3, jsonjq=4, script=5)
	case "request-header":
		return 200
	case "request-body":
		return 203
	case "jsonjq-request-body":
		return 204
	case "script-request-header":
		return 205
	case "script-request-body":
		return 205

	// Group 3: response (sub: header=0, echo=1, analyze=2, body=3, jsonjq=4, script=5)
	case "response-header":
		return 300
	case "echo-response":
		return 301
	case "script-echo-response":
		return 301
	case "script-analyze-echo-response":
		return 302
	case "response-body":
		return 303
	case "jsonjq-response-body":
		return 304
	case "script-response-header":
		return 305
	case "script-response-body":
		return 305

	// bare header → request header
	case "header":
		return 200

	default:
		return 400
	}
}

func isRejectFamilyAction(action string) bool {
	switch action {
	case "reject", "reject-200", "reject-img", "reject-dict", "reject-array":
		return true
	default:
		return false
	}
}

func parseRewriteLine(line string) RewriteRule {
	r := RewriteRule{Line: line}

	if strings.HasPrefix(line, "hostname") {
		r.Cat = "hostname"
		r.Pattern = line
		return r
	}

	sep := " url "
	sepIdx := strings.Index(line, sep)
	isURLAndHeader := false
	if sepIdx < 0 {
		sep = " url-and-header "
		sepIdx = strings.Index(line, sep)
		isURLAndHeader = sepIdx >= 0
	}

	if sepIdx >= 0 {
		r.Pattern = line[:sepIdx]
		actionPart := line[sepIdx+len(sep):]

		actionFields := strings.SplitN(actionPart, " ", 2)
		action := strings.ToLower(actionFields[0])
		r.Cat = action

		if isURLAndHeader && isRejectFamilyAction(action) {
			r.Cat = "reject"
		}

		normalizedActionPart := action
		if len(actionFields) > 1 {
			r.SubType = actionFields[1]
			normalizedActionPart += " " + actionFields[1]
		}
		r.Line = r.Pattern + sep + normalizedActionPart
		return r
	}

	if idx := strings.Index(line, " - "); idx >= 0 {
		r.Pattern = line[:idx]
		actionPart := line[idx+3:]
		action := strings.ToLower(strings.TrimSpace(actionPart))
		r.Cat = action
		r.Line = r.Pattern + " url " + action
		return r
	}

	fields := strings.Fields(line)
	if len(fields) >= 2 {
		r.Pattern = fields[0]
		action := strings.ToLower(fields[1])
		r.Cat = action
		if len(fields) > 2 {
			r.SubType = strings.Join(fields[2:], " ")
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