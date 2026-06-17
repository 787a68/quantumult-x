package transform

import "strings"

func TransformQxLine(line string) (string, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return "", errUnrecognized
	}
	typePart := strings.TrimSpace(parts[0])
	valuePart := strings.TrimSpace(parts[1])
	switch strings.ToLower(typePart) {
	case "host-suffix", "host-keyword", "host", "host-wildcard",
		"ip-cidr", "ip6-cidr", "ip-asn", "geoip":
		return strings.ToLower(typePart) + "," + valuePart, nil
	default:
		return "", errUnrecognized
	}
}