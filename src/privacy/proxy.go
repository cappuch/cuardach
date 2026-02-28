package privacy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/cappuch/cuardach/src/config"
	"golang.org/x/net/proxy"
)

func configureProxy(transport *http.Transport, cfg config.ProxyConfig) error {
	proxyURL, err := url.Parse(cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid proxy address %q: %w", cfg.Address, err)
	}

	switch cfg.Type {
	case "socks5":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return fmt.Errorf("creating SOCKS5 dialer: %w", err)
		}
		transport.DialContext = func(_ context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
	default:
		return fmt.Errorf("unsupported proxy type %q (use socks5 or http)", cfg.Type)
	}

	return nil
}
