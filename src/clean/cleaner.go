package clean

import (
	"bufio"
	"io"
	"strings"

	"github.com/quantumult-x/gen/src/log"
)

func CleanLines(r io.Reader) <-chan string {
	ch := make(chan string, 1024)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
				continue
			}
			line = stripTrailingComment(line)
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			ch <- line
		}
		if err := scanner.Err(); err != nil {
			log.Warn("cleanlines: scanner error: %v", err)
		}
	}()
	return ch
}

func stripTrailingComment(line string) string {
	inURL := false
	for i := 0; i < len(line); i++ {
		if i > 0 && line[i-1] == ':' && line[i] == '/' && i+1 < len(line) && line[i+1] == '/' {
			inURL = true
			i += 2
			continue
		}
		if !inURL && i+1 < len(line) && line[i] == ' ' && (line[i+1] == '#' || (line[i+1] == '/' && i+2 < len(line) && line[i+2] == '/')) {
			return line[:i]
		}
		if inURL && line[i] == ' ' {
			inURL = false
		}
	}
	return line
}
