package dedup

import "sort"

func TextDedup(lines []string) ([]string, int) {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		result = append(result, line)
	}
	return result, len(lines) - len(result)
}

func DedupSorted(items []string) []string {
	deduped, _ := TextDedup(items)
	sort.Strings(deduped)
	return deduped
}
