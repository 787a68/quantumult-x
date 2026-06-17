package dedup

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