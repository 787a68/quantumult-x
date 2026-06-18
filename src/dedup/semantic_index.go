package dedup

import "strings"

type SemanticIndex struct {
	suffixSet  map[string]struct{}
	hostSet    map[string]struct{}
	keywords   []string
	ipCidrs    []string
	ip6Cidrs   []string
	ipASNs     map[string]struct{}
	wildcards  map[string]struct{}
}

func NewSemanticIndex() *SemanticIndex {
	return &SemanticIndex{
		suffixSet: make(map[string]struct{}),
		hostSet:   make(map[string]struct{}),
		keywords:  make([]string, 0),
		ipCidrs:   make([]string, 0),
		ip6Cidrs:  make([]string, 0),
		ipASNs:    make(map[string]struct{}),
		wildcards: make(map[string]struct{}),
	}
}

func (si *SemanticIndex) Add(r Rule) {
	switch r.Type {
	case "host-suffix":
		si.suffixSet[r.Value] = struct{}{}
	case "host":
		si.hostSet[r.Value] = struct{}{}
	case "host-keyword":
		si.keywords = append(si.keywords, r.Value)
	case "host-wildcard":
		si.wildcards[r.Value] = struct{}{}
	case "ip-cidr":
		si.ipCidrs = append(si.ipCidrs, r.Value)
	case "ip6-cidr":
		si.ip6Cidrs = append(si.ip6Cidrs, r.Value)
	case "ip-asn":
		si.ipASNs[r.Value] = struct{}{}
	}
}

func (si *SemanticIndex) IsCovered(r Rule) bool {
	switch r.Type {
	case "host-suffix":
		if _, ok := si.suffixSet[r.Value]; ok {
			return true
		}
		parts := strings.Split(r.Value, ".")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[i:], ".")
			if _, ok := si.suffixSet[parent]; ok {
				return true
			}
		}
		for _, kw := range si.keywords {
			if strings.Contains(r.Value, kw) {
				return true
			}
		}
		return false
	case "host":
		if _, ok := si.hostSet[r.Value]; ok {
			return true
		}
		parts := strings.Split(r.Value, ".")
		for i := 1; i < len(parts); i++ {
			suffix := "." + strings.Join(parts[i:], ".")
			if _, ok := si.suffixSet[suffix[1:]]; ok {
				return true
			}
		}
		for _, kw := range si.keywords {
			if strings.Contains(r.Value, kw) {
				return true
			}
		}
		return false
	case "host-keyword":
		for _, kw := range si.keywords {
			if strings.Contains(r.Value, kw) {
				return true
			}
		}
		return false
	case "ip-cidr":
		for _, kw := range si.keywords {
			if strings.Contains(r.Value, kw) {
				return true
			}
		}
		return false
	case "ip6-cidr", "ip-asn":
		return false
	}
	return false
}