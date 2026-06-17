package fetch

import (
	"io"
	"sync"
	"time"

	"github.com/quantumult-x/gen/src/log"
	"github.com/quantumult-x/gen/src/util"
)

func FetchAll(sources []util.UpstreamSource, concurrency int, timeout time.Duration, retries int) (map[string]io.ReadCloser, map[string]error) {
	results := make(map[string]io.ReadCloser)
	errors := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, concurrency)

	for _, src := range sources {
		wg.Add(1)
		go func(s util.UpstreamSource) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var reader io.ReadCloser
			var err error
			for attempt := 0; attempt <= retries; attempt++ {
				if attempt > 0 {
					log.Warn("retry %d/%d for %s", attempt, retries, s.URL)
					time.Sleep(time.Duration(attempt) * time.Second)
				}
				reader, err = GetReader(s.URL, timeout)
				if err == nil {
					break
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				log.Warn("failed to fetch %s: %v", s.URL, err)
				errors[s.URL] = err
			} else {
				results[s.URL] = reader
			}
		}(src)
	}

	wg.Wait()
	return results, errors
}