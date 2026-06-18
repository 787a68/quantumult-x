package transform

import (
	"regexp"
)

var (
	reDomainSuffix  = regexp.MustCompile(`(?i)^DOMAIN-SUFFIX\s*,\s*([^,]+)`)
	reDomainKeyword = regexp.MustCompile(`(?i)^DOMAIN-KEYWORD\s*,\s*([^,]+)`)
	reDomain        = regexp.MustCompile(`(?i)^DOMAIN\s*,\s*([^,]+)`)
	reIPCidr        = regexp.MustCompile(`(?i)^IP-CIDR\s*,\s*([^,]+)`)
	reIPCidr6       = regexp.MustCompile(`(?i)^IP-CIDR6\s*,\s*([^,]+)`)
	reIPASN         = regexp.MustCompile(`(?i)^IP-ASN\s*,\s*([^,]+)`)
)

func TransformRuleLine(line string) (string, error) {
	if m := reDomainSuffix.FindStringSubmatch(line); m != nil {
		return "host-suffix," + m[1], nil
	}
	if m := reDomainKeyword.FindStringSubmatch(line); m != nil {
		return "host-keyword," + m[1], nil
	}
	if m := reDomain.FindStringSubmatch(line); m != nil {
		return "host," + m[1], nil
	}
	if m := reIPCidr.FindStringSubmatch(line); m != nil {
		return "ip-cidr," + m[1], nil
	}
	if m := reIPCidr6.FindStringSubmatch(line); m != nil {
		return "ip6-cidr," + m[1], nil
	}
	if m := reIPASN.FindStringSubmatch(line); m != nil {
		return "ip-asn," + m[1], nil
	}
	return "", errUnrecognized
}
