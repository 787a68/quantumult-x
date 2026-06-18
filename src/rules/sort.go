package rules

import (
	"sort"
	"strings"
)

func typeOrder(ruleType string) int {
	switch ruleType {
	case "host-keyword":
		return 0
	case "host-wildcard":
		return 1
	case "host-suffix":
		return 2
	case "host":
		return 3
	case "ip-cidr":
		return 4
	case "ip6-cidr":
		return 5
	case "ip-asn":
		return 6
	case "geoip":
		return 7
	default:
		return 8
	}
}

func Sort(lines []string) []string {
	typeGroups := map[int][]string{}
	var keys []int
	for _, line := range lines {
		o := typeOrder(strings.SplitN(line, ",", 2)[0])
		if _, exists := typeGroups[o]; !exists {
			keys = append(keys, o)
		}
		typeGroups[o] = append(typeGroups[o], line)
	}
	sort.Ints(keys)

	var result []string
	for _, k := range keys {
		group := typeGroups[k]
		sort.SliceStable(group, func(i, j int) bool {
			return domainSortKey(group[i]) < domainSortKey(group[j])
		})
		result = append(result, group...)
	}
	return result
}

func domainSortKey(line string) string {
	parts := strings.SplitN(line, ",", 2)
	if len(parts) < 2 {
		return line
	}
	ruleType := parts[0]
	value := parts[1]

	switch ruleType {
	case "host-suffix", "host":
		domains := strings.Split(value, ".")
		reversed := make([]string, len(domains))
		for i, d := range domains {
			reversed[len(domains)-1-i] = d
		}
		return strings.Join(reversed, ".")
	default:
		return value
	}
}
