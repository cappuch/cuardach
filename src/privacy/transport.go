package privacy

import (
	"net"
	"net/http"
	"time"

	"github.com/cappuch/cuardach/src/config"
)

type privacyTransport struct {
	base     http.RoundTripper
	rotator  *UserAgentRotator
	pool     *ProxyPool
	stripRef bool
}

func (t *privacyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.stripRef {
		req.Header.Del("Referer")
	}

	req.Header.Del("Cookie")

	if t.rotator != nil {
		req.Header.Set("User-Agent", t.rotator.Get())
	}

	req.Header.Set("DNT", "1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	if t.pool != nil {
		if proxy := t.pool.Get(); proxy != nil {
			if baseT, ok := t.base.(*http.Transport); ok {
				rt := baseT.Clone()
				rt.Proxy = http.ProxyURL(proxy.URL)
				resp, err := rt.RoundTrip(req)
				if err != nil {
					t.pool.Remove(proxy.Address)
				} else {
					return resp, nil
				}
			}
		}
	}

	return t.base.RoundTrip(req)
}

func NewClient(cfg config.PrivacyConfig, pool *ProxyPool) (*http.Client, error) {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		MaxIdleConns:          50,
		IdleConnTimeout:       30 * time.Second,
	}

	if cfg.Proxy.Enabled {
		if err := configureProxy(transport, cfg.Proxy); err != nil {
			return nil, err
		}
	}

	var rotator *UserAgentRotator
	if cfg.UserAgentRotation {
		rotator = NewUserAgentRotator()
	}

	var activePool *ProxyPool
	if !cfg.Proxy.Enabled && pool != nil && pool.Count() > 0 {
		activePool = pool
	}

	pt := &privacyTransport{
		base:     transport,
		rotator:  rotator,
		pool:     activePool,
		stripRef: cfg.StripReferrer,
	}

	return &http.Client{
		Transport: pt,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			req.Header.Del("Cookie")
			req.Header.Del("Referer")
			return nil
		},
	}, nil
}
