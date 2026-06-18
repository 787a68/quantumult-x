package dedup

import "strings"

type SemanticIndex struct {
	suffixSet  map[string]struct{}
	hostSet    map[string]struct{}
	keywords   []string
	ipCidrSet  map[string]struct{}
	ip6CidrSet map[string]struct{}
	ipASNSet   map[string]struct{}
	geoipSet   map[string]struct{}
	wildcards  map[string]struct{}
}

func NewSemanticIndex() *SemanticIndex {
	return &SemanticIndex{
		suffixSet:  make(map[string]struct{}),
		hostSet:    make(map[string]struct{}),
		ipCidrSet:  make(map[string]struct{}),
		ip6CidrSet: make(map[string]struct{}),
		ipASNSet:   make(map[string]struct{}),
		geoipSet:   make(map[string]struct{}),
		wildcards:  make(map[string]struct{}),
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
		si.ipCidrSet[r.Value] = struct{}{}
	case "ip6-cidr":
		si.ip6CidrSet[r.Value] = struct{}{}
	case "ip-asn":
		si.ipASNSet[r.Value] = struct{}{}
	case "geoip":
		si.geoipSet[r.Value] = struct{}{}
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
			if _, ok := si.suffixSet[strings.Join(parts[i:], ".")]; ok {
				return true
			}
		}
		return si.coveredByKeyword(r.Value)
	case "host":
		if _, ok := si.hostSet[r.Value]; ok {
			return true
		}
		parts := strings.Split(r.Value, ".")
		for i := 1; i < len(parts); i++ {
			if _, ok := si.suffixSet[strings.Join(parts[i:], ".")]; ok {
				return true
			}
		}
		return si.coveredByKeyword(r.Value)
	case "host-keyword":
		return si.coveredByKeyword(r.Value)
	case "host-wildcard":
		if _, ok := si.wildcards[r.Value]; ok {
			return true
		}
		return false
	case "ip-cidr":
		if _, ok := si.ipCidrSet[r.Value]; ok {
			return true
		}
		return false
	case "ip6-cidr":
		if _, ok := si.ip6CidrSet[r.Value]; ok {
			return true
		}
		return false
	case "ip-asn":
		if _, ok := si.ipASNSet[r.Value]; ok {
			return true
		}
		return false
	case "geoip":
		if _, ok := si.geoipSet[r.Value]; ok {
			return true
		}
		return false
	}
	return false
}

func (si *SemanticIndex) coveredByKeyword(value string) bool {
	for _, kw := range si.keywords {
		if strings.Contains(value, kw) {
			return true
		}
	}
	return false
}
