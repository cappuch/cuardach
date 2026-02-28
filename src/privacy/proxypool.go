package privacy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Proxy struct {
	Address  string // host:port
	Type     string // "http", "socks5"
	URL      *url.URL
	Latency  time.Duration
	LastSeen time.Time
}

type ProxyPool struct {
	mu        sync.RWMutex
	proxies   []Proxy
	sources   []string
	testURL   string
	timeout   time.Duration
	minCount  int
	cachePath string
	cacheTTL  time.Duration
}

type cachedProxies struct {
	SavedAt time.Time     `json:"saved_at"`
	Proxies []cachedEntry `json:"proxies"`
}

type cachedEntry struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Latency int64  `json:"latency_ms"`
}

var defaultProxySources = []string{
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
	"https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/http.txt",
	"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt",
	"https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt",
	"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt",
}

func NewProxyPool() *ProxyPool {
	home, _ := os.UserHomeDir()
	return &ProxyPool{
		sources:   defaultProxySources,
		testURL:   "http://httpbin.org/ip",
		timeout:   5 * time.Second,
		minCount:  3,
		cachePath: filepath.Join(home, ".cache", "cuardach", "proxies.json"),
		cacheTTL:  1 * time.Hour,
	}
}

func (p *ProxyPool) Load(ctx context.Context, onProgress func(tested, working int)) (int, error) {
	if n, ok := p.loadCache(); ok {
		return n, nil
	}
	return p.loadFresh(ctx, onProgress)
}

func (p *ProxyPool) loadFresh(ctx context.Context, onProgress func(tested, working int)) (int, error) {
	raw := p.fetchAll(ctx)
	if len(raw) == 0 {
		return 0, fmt.Errorf("no proxies fetched from any source")
	}

	rand.Shuffle(len(raw), func(i, j int) { raw[i], raw[j] = raw[j], raw[i] })
	if len(raw) > 500 {
		raw = raw[:500]
	}

	type result struct {
		proxy Proxy
		ok    bool
	}
	ch := make(chan result, len(raw))
	sem := make(chan struct{}, 100)

	var wg sync.WaitGroup
	tested := 0
	working := 0
	var mu sync.Mutex

	for _, r := range raw {
		wg.Add(1)
		go func(addr, typ string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			proxy, ok := p.validate(ctx, addr, typ)

			mu.Lock()
			tested++
			if ok {
				working++
			}
			t, w := tested, working
			mu.Unlock()

			if onProgress != nil {
				onProgress(t, w)
			}

			ch <- result{proxy: proxy, ok: ok}
		}(r.addr, r.typ)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var valid []Proxy
	for res := range ch {
		if res.ok {
			valid = append(valid, res.proxy)
		}
	}

	p.mu.Lock()
	p.proxies = valid
	p.mu.Unlock()

	p.saveCache(valid)

	return len(valid), nil
}

func (p *ProxyPool) loadCache() (int, bool) {
	data, err := os.ReadFile(p.cachePath)
	if err != nil {
		return 0, false
	}

	var cached cachedProxies
	if err := json.Unmarshal(data, &cached); err != nil {
		return 0, false
	}

	if time.Since(cached.SavedAt) > p.cacheTTL {
		return 0, false
	}

	if len(cached.Proxies) == 0 {
		return 0, false
	}

	var proxies []Proxy
	for _, e := range cached.Proxies {
		scheme := "http://"
		if e.Type == "socks5" {
			scheme = "socks5://"
		}
		u, err := url.Parse(scheme + e.Address)
		if err != nil {
			continue
		}
		proxies = append(proxies, Proxy{
			Address:  e.Address,
			Type:     e.Type,
			URL:      u,
			Latency:  time.Duration(e.Latency) * time.Millisecond,
			LastSeen: cached.SavedAt,
		})
	}

	p.mu.Lock()
	p.proxies = proxies
	p.mu.Unlock()

	return len(proxies), true
}

func (p *ProxyPool) saveCache(proxies []Proxy) {
	var entries []cachedEntry
	for _, px := range proxies {
		entries = append(entries, cachedEntry{
			Address: px.Address,
			Type:    px.Type,
			Latency: px.Latency.Milliseconds(),
		})
	}

	cached := cachedProxies{
		SavedAt: time.Now(),
		Proxies: entries,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(p.cachePath), 0700)
	os.WriteFile(p.cachePath, data, 0600)
}

type rawProxy struct {
	addr string
	typ  string
}

func (p *ProxyPool) fetchAll(ctx context.Context) []rawProxy {
	client := &http.Client{Timeout: 15 * time.Second}
	seen := make(map[string]bool)
	var all []rawProxy
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, src := range p.sources {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			typ := "http"
			if strings.Contains(u, "socks5") || strings.Contains(u, "socks") {
				typ = "socks5"
			}

			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			scanner := bufio.NewScanner(io.LimitReader(resp.Body, 2<<20))
			mu.Lock()
			defer mu.Unlock()
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if !strings.Contains(line, ":") {
					continue
				}
				if seen[line] {
					continue
				}
				seen[line] = true
				all = append(all, rawProxy{addr: line, typ: typ})
			}
		}(src)
	}

	wg.Wait()
	return all
}

func (p *ProxyPool) validate(ctx context.Context, addr, typ string) (Proxy, bool) {
	var proxyURL *url.URL
	var err error

	switch typ {
	case "socks5":
		proxyURL, err = url.Parse("socks5://" + addr)
	default:
		proxyURL, err = url.Parse("http://" + addr)
		typ = "http"
	}
	if err != nil {
		return Proxy{}, false
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: p.timeout}).DialContext,
		TLSHandshakeTimeout:   p.timeout,
		ResponseHeaderTimeout: p.timeout,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   p.timeout,
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", p.testURL, nil)
	if err != nil {
		return Proxy{}, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return Proxy{}, false
	}
	defer resp.Body.Close()
	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return Proxy{}, false
	}

	body := make([]byte, 512)
	n, _ := io.ReadAtLeast(resp.Body, body, 1)
	if n == 0 {
		return Proxy{}, false
	}

	return Proxy{
		Address:  addr,
		Type:     typ,
		URL:      proxyURL,
		Latency:  latency,
		LastSeen: time.Now(),
	}, true
}

func (p *ProxyPool) Get() *Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.proxies) == 0 {
		return nil
	}
	proxy := p.proxies[rand.Intn(len(p.proxies))]
	return &proxy
}

func (p *ProxyPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

func (p *ProxyPool) List() []Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]Proxy, len(p.proxies))
	copy(out, p.proxies)
	return out
}

func (p *ProxyPool) Remove(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, px := range p.proxies {
		if px.Address == addr {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			return
		}
	}
}
