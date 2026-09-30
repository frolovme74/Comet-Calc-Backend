package media

import (
	"net/http"
	"sync"
	"time"
)

const (
	DefaultCometPhoto = "/static/media/default_comet.jpg"
	DefaultCometVideo = "/static/media/default_comet.mp4"
)

type Checker struct {
	client *http.Client
	ttl    time.Duration
	mu     sync.Mutex
	cache  map[string]checkResult
}

type checkResult struct {
	ok      bool
	checked time.Time
}

func NewChecker() *Checker {
	return &Checker{
		client: &http.Client{Timeout: 700 * time.Millisecond},
		ttl:    30 * time.Second,
		cache:  map[string]checkResult{},
	}
}

func (c *Checker) available(url string) bool {
	c.mu.Lock()
	res, found := c.cache[url]
	c.mu.Unlock()
	if found && time.Since(res.checked) < c.ttl {
		return res.ok
	}

	ok := false
	if resp, err := c.client.Head(url); err == nil {
		resp.Body.Close()
		ok = resp.StatusCode == http.StatusOK
	}

	c.mu.Lock()
	c.cache[url] = checkResult{ok: ok, checked: time.Now()}
	c.mu.Unlock()
	return ok
}

func (c *Checker) Photo(url string) string {
	if url == "" || !c.available(url) {
		return DefaultCometPhoto
	}
	return url
}

func (c *Checker) Video(url string) string {
	if url == "" || !c.available(url) {
		return DefaultCometVideo
	}
	return url
}
