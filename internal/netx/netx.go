// Package netx resolves the real client address behind (optional) trusted reverse proxies.
package netx

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// Trusted is a set of proxy addresses/networks. The zero value trusts nobody.
type Trusted struct {
	prefixes []netip.Prefix
}

// ParseTrusted parses a list of IPs and CIDRs ("10.0.0.0/8", "::1", "172.18.0.2").
func ParseTrusted(entries []string) (*Trusted, error) {
	t := &Trusted{}
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.Contains(e, "/") {
			p, err := netip.ParsePrefix(e)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", e, err)
			}
			t.prefixes = append(t.prefixes, normalize(p))
			continue
		}
		a, err := netip.ParseAddr(e)
		if err != nil {
			return nil, fmt.Errorf("invalid IP %q: %w", e, err)
		}
		a = a.Unmap()
		t.prefixes = append(t.prefixes, netip.PrefixFrom(a, a.BitLen()))
	}
	return t, nil
}

// normalize turns an IPv4-mapped IPv6 prefix into the equivalent IPv4 prefix and masks host bits.
func normalize(p netip.Prefix) netip.Prefix {
	a := p.Addr()
	if a.Is4In6() && p.Bits() >= 96 {
		return netip.PrefixFrom(a.Unmap(), p.Bits()-96).Masked()
	}
	return p.Masked()
}

// Empty reports whether no proxy is trusted.
func (t *Trusted) Empty() bool { return t == nil || len(t.prefixes) == 0 }

// Contains reports whether addr belongs to a trusted proxy.
func (t *Trusted) Contains(addr netip.Addr) bool {
	if t == nil {
		return false
	}
	addr = addr.Unmap().WithZone("")
	for _, p := range t.prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// Result is the outcome of resolving a request's client address.
type Result struct {
	// IP is the resolved client address; invalid only when the peer address itself is unparsable.
	IP netip.Addr
	// UntrustedForwarded is set when a peer that is not a trusted proxy sent X-Forwarded-For
	// (the header was ignored; rate limits see the peer, which is probably a misconfigured proxy).
	UntrustedForwarded bool
	// Malformed is set when a trusted proxy sent a forwarded chain that could not be used.
	Malformed bool
}

// PeerAddr extracts the IP from a "host:port" RemoteAddr.
func PeerAddr(remoteAddr string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return ap.Addr().Unmap().WithZone(""), true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap().WithZone(""), true
}

// ClientIP applies the baseline §11.4 rules: unless the peer is a trusted proxy the peer address is the
// client; otherwise X-Forwarded-For (all header lines, in order) is walked right to left skipping trusted
// hops, and the first untrusted entry is the client. An empty or unparsable chain falls back to the peer.
func (t *Trusted) ClientIP(remoteAddr string, xff []string) Result {
	peer, ok := PeerAddr(remoteAddr)
	hasXFF := false
	for _, v := range xff {
		if strings.TrimSpace(v) != "" {
			hasXFF = true
		}
	}
	if !ok {
		return Result{Malformed: hasXFF}
	}
	if !t.Contains(peer) {
		return Result{IP: peer, UntrustedForwarded: hasXFF}
	}
	if !hasXFF {
		return Result{IP: peer}
	}
	var chain []string
	for _, v := range xff {
		chain = append(chain, strings.Split(v, ",")...)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		entry := strings.TrimSpace(chain[i])
		if entry == "" {
			return Result{IP: peer, Malformed: true}
		}
		a, ok := parseForwarded(entry)
		if !ok {
			return Result{IP: peer, Malformed: true}
		}
		if !t.Contains(a) {
			return Result{IP: a}
		}
	}
	// Every hop was a trusted proxy: the client is the left-most entry's address, which is trusted too.
	a, _ := parseForwarded(strings.TrimSpace(chain[0]))
	return Result{IP: a}
}

// parseForwarded accepts "1.2.3.4", "1.2.3.4:5678", "::1", "[::1]" and "[::1]:80".
func parseForwarded(s string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap().WithZone(""), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap().WithZone(""), true
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		if a, err := netip.ParseAddr(s[1 : len(s)-1]); err == nil {
			return a.Unmap().WithZone(""), true
		}
	}
	return netip.Addr{}, false
}

// IsLoopbackListen reports whether a listen address ("host:port") only accepts local connections.
func IsLoopbackListen(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	a, err := netip.ParseAddr(host)
	return err == nil && a.IsLoopback()
}
