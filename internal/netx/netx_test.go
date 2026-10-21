package netx

import "testing"

func mustTrusted(t *testing.T, entries ...string) *Trusted {
	t.Helper()
	tr, err := ParseTrusted(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func TestClientIP(t *testing.T) {
	proxies := mustTrusted(t, "10.0.0.0/8", "::1", "192.168.1.5", "fd00::/8")
	tests := []struct {
		name      string
		trusted   *Trusted
		remote    string
		xff       []string
		want      string
		untrusted bool
		malformed bool
	}{
		{"no proxies, no header", mustTrusted(t), "203.0.113.9:5555", nil, "203.0.113.9", false, false},
		{"untrusted peer spoofing xff", proxies, "203.0.113.9:5555", []string{"1.1.1.1"}, "203.0.113.9", true, false},
		{"empty trusted list ignores xff", mustTrusted(t), "127.0.0.1:1", []string{"8.8.8.8"}, "127.0.0.1", true, false},
		{"trusted peer, single hop", proxies, "10.1.2.3:1", []string{"198.51.100.7"}, "198.51.100.7", false, false},
		{"right to left skips trusted", proxies, "10.0.0.1:1", []string{"6.6.6.6, 198.51.100.7, 10.9.9.9"}, "198.51.100.7", false, false},
		{"client-forged left entries are ignored", proxies, "10.0.0.1:1", []string{"1.2.3.4", "198.51.100.7"}, "198.51.100.7", false, false},
		{"all hops trusted", proxies, "10.0.0.1:1", []string{"10.2.2.2, 192.168.1.5"}, "10.2.2.2", false, false},
		{"trusted peer without xff", proxies, "10.0.0.1:1", nil, "10.0.0.1", false, false},
		{"ipv6 peer trusted", proxies, "[::1]:4000", []string{"2001:db8::7"}, "2001:db8::7", false, false},
		{"ipv6 in brackets with port", proxies, "10.0.0.1:1", []string{"[2001:db8::8]:443"}, "2001:db8::8", false, false},
		{"ipv4 with port", proxies, "10.0.0.1:1", []string{"198.51.100.1:9999"}, "198.51.100.1", false, false},
		{"ipv4-mapped ipv6 peer", proxies, "[::ffff:10.0.0.9]:1", []string{"198.51.100.2"}, "198.51.100.2", false, false},
		{"garbage entry falls back to peer", proxies, "10.0.0.1:1", []string{"not-an-ip"}, "10.0.0.1", false, true},
		{"garbage after valid hop falls back", proxies, "10.0.0.1:1", []string{"198.51.100.2, garbage"}, "10.0.0.1", false, true},
		{"empty entry", proxies, "10.0.0.1:1", []string{"1.1.1.1,,10.0.0.2"}, "10.0.0.1", false, true},
		{"whitespace only header is no header", proxies, "10.0.0.1:1", []string{"  "}, "10.0.0.1", false, false},
		{"untrusted ipv6 peer", proxies, "[2001:db8::1]:80", nil, "2001:db8::1", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.trusted.ClientIP(tc.remote, tc.xff)
			if got.IP.String() != tc.want {
				t.Errorf("IP = %v, want %s", got.IP, tc.want)
			}
			if got.UntrustedForwarded != tc.untrusted {
				t.Errorf("UntrustedForwarded = %v, want %v", got.UntrustedForwarded, tc.untrusted)
			}
			if got.Malformed != tc.malformed {
				t.Errorf("Malformed = %v, want %v", got.Malformed, tc.malformed)
			}
		})
	}
}

func TestUnparsablePeer(t *testing.T) {
	var tr *Trusted
	r := tr.ClientIP("garbage", nil)
	if r.IP.IsValid() {
		t.Fatalf("expected invalid IP, got %v", r.IP)
	}
}

func TestParseTrustedErrors(t *testing.T) {
	for _, bad := range []string{"10.0.0.0/33", "abc", "1.2.3", "::g"} {
		if _, err := ParseTrusted([]string{bad}); err == nil {
			t.Errorf("ParseTrusted(%q) succeeded", bad)
		}
	}
	if tr, err := ParseTrusted([]string{"", "  "}); err != nil || !tr.Empty() {
		t.Fatalf("blank entries: %v %v", tr, err)
	}
}

func TestIsLoopbackListen(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:12433": true, "[::1]:80": true, "localhost:80": true,
		"0.0.0.0:12433": false, ":12433": false, "[::]:80": false, "192.168.1.2:80": false, "garbage": false,
	} {
		if got := IsLoopbackListen(addr); got != want {
			t.Errorf("IsLoopbackListen(%q) = %v, want %v", addr, got, want)
		}
	}
}
