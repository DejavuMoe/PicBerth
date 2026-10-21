package ids

import (
	"regexp"
	"strings"
	"testing"
)

var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

func TestImageIDShape(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-hjkmnp-tv-z]{20}$`)
	seen := map[string]bool{}
	for i := 0; i < 5000; i++ {
		id := NewImageID()
		if !re.MatchString(id) {
			t.Fatalf("bad id %q", id)
		}
		if !ValidImageID(id) {
			t.Fatalf("ValidImageID rejects %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestValidImageIDRejects(t *testing.T) {
	for _, s := range []string{"", "short", strings.Repeat("a", 21), strings.Repeat("A", 20), strings.Repeat("i", 20),
		strings.Repeat("l", 20), strings.Repeat("o", 20), strings.Repeat("u", 20), "../../../etc/passwd0000", strings.Repeat("a", 19) + "/"} {
		if ValidImageID(s) {
			t.Errorf("ValidImageID(%q) = true", s)
		}
	}
}

// Property: whatever the ID, the object key only contains [0-9a-z./] and no path segment
// (with or without extension) is a Windows reserved device name.
func TestObjectKeyProperty(t *testing.T) {
	charset := regexp.MustCompile(`^[0-9a-z./]+$`)
	for i := 0; i < 20000; i++ {
		id := NewImageID()
		for _, ext := range []string{"jpg", "png", "gif", "webp", "avif"} {
			key := ObjectKey(id, ext)
			if !charset.MatchString(key) {
				t.Fatalf("key %q has forbidden characters", key)
			}
			if len(key) > 40 {
				t.Fatalf("key %q too long", key)
			}
			for _, seg := range strings.Split(key, "/") {
				base := seg
				if dot := strings.IndexByte(base, '.'); dot >= 0 {
					base = base[:dot]
				}
				if windowsReserved[strings.ToLower(base)] {
					t.Fatalf("segment %q of %q is a reserved name", seg, key)
				}
				if strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " ") {
					t.Fatalf("segment %q ends with dot or space", seg)
				}
			}
		}
	}
}

func TestTokenSecret(t *testing.T) {
	s := NewTokenSecret()
	if !ValidTokenSecret(s) {
		t.Fatalf("invalid secret %q", s)
	}
	if len(s) != 35 || !strings.HasPrefix(s, "pb_") {
		t.Fatalf("unexpected shape %q", s)
	}
	if got := TokenDisplay(s); got != s[:9] || len(got) != 9 {
		t.Fatalf("display %q", got)
	}
	if ValidTokenSecret("pb_short") || ValidTokenSecret(strings.ToUpper(s)) {
		t.Fatal("accepted malformed secret")
	}
}

func TestSetupCode(t *testing.T) {
	re := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)
	c := NewSetupCode()
	if !re.MatchString(c) {
		t.Fatalf("bad setup code %q", c)
	}
	if NormalizeSetupCode(" "+strings.ToLower(c)+" ") != strings.ReplaceAll(c, "-", "") {
		t.Fatal("normalize failed")
	}
}

func TestSessionIDAndHash(t *testing.T) {
	a, b := NewSessionID(), NewSessionID()
	if a == b || len(a) != 43 {
		t.Fatalf("session ids %q %q", a, b)
	}
	if h := Hash("x"); len(h) != 64 || h != Hash("x") || h == Hash("y") {
		t.Fatal("hash misbehaves")
	}
}
