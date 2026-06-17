package fetch

import (
	"io"
	"net/http"
	"time"

	"github.com/quantumult-x/gen/src/log"
)

func GetReader(url string, timeout time.Duration) (io.ReadCloser, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "quantumult-x-gen/1.0")
	log.Debug("fetching %s", url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, errHTTP{status: resp.StatusCode, url: url}
	}
	return resp.Body, nil
}

type errHTTP struct {
	status int
	url    string
}

func (e errHTTP) Error() string {
	return "HTTP " + string(rune(e.status)) + " for " + e.url
}