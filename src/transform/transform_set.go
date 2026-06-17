package transform

import "strings"

func TransformSetLine(line string) (string, error) {
	value := strings.TrimSpace(line)
	if strings.HasPrefix(value, ".") {
		value = strings.TrimPrefix(value, ".")
	}
	if value == "" {
		return "", errUnrecognized
	}
	return "host-suffix," + value, nil
}