package clean

import (
	"bufio"
	"os"
	"strings"

	"github.com/quantumult-x/gen/src/util"
)

type ConfSpec struct {
	HeadRules []string
	Upstreams []util.UpstreamSource
	Excludes  []string
}

func ParseConf(path string) (*ConfSpec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	spec := &ConfSpec{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			exc := strings.TrimPrefix(line, "-")
			spec.Excludes = append(spec.Excludes, exc)
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			src := parseUpstream(line)
			spec.Upstreams = append(spec.Upstreams, src)
			continue
		}
		spec.HeadRules = append(spec.HeadRules, line)
	}
	return spec, scanner.Err()
}

func parseUpstream(line string) util.UpstreamSource {
	parts := strings.SplitN(line, ",", 2)
	src := util.UpstreamSource{URL: parts[0]}
	if len(parts) == 2 {
		src.Format = strings.TrimSpace(parts[1])
	} else {
		src.Format = "qx"
	}
	return src
}

func (s *ConfSpec) InsertHeadRules(lines []string) []string {
	if len(s.HeadRules) == 0 {
		return lines
	}
	return append(s.HeadRules, lines...)
}

func (s *ConfSpec) ApplyExcludes(lines []string) []string {
	if len(s.Excludes) == 0 {
		return lines
	}
	var result []string
	for _, line := range lines {
		excluded := false
		for _, exc := range s.Excludes {
			if strings.HasPrefix(line, exc) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, line)
		}
	}
	return result
}
