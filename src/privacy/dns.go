package privacy

// dns-over-https support is available via proxy configuration.
// when using a SOCKS5 proxy (e.g., Tor), dns resolution is performed
// remotely by the proxy, preventing dns leaks.
//
// For standalone doh without a proxy, configure your system resolver
// or use a local doh client like dnscrypt-proxy.
