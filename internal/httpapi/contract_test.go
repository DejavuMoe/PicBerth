package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"testing"
)

// The ETag must change with the document: a client holding an older contract has to get the new one.
func TestOpenAPIETagFollowsDocument(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.do("GET", "/api/v1/openapi.json", nil)
	wantStatus(t, resp, body, 200)
	sum := sha256.Sum256(body)
	want := `"` + hex.EncodeToString(sum[:]) + `"`
	if got := resp.Header.Get("ETag"); got != want {
		t.Fatalf("ETag %s does not identify the served document (want %s)", got, want)
	}
	resp, body = h.do("GET", "/api/v1/openapi.json", nil, withHeader("If-None-Match", want))
	if resp.StatusCode != 304 || len(body) != 0 {
		t.Fatalf("revalidation: %d with %d bytes", resp.StatusCode, len(body))
	}
}

// Every problem code the server can send is part of the published contract.
func TestOpenAPIListsEveryProblemCode(t *testing.T) {
	var doc struct {
		Components struct {
			Schemas struct {
				Problem struct {
					Properties struct {
						Code struct {
							Enum []string `json:"enum"`
						} `json:"code"`
					} `json:"properties"`
				} `json:"Problem"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(openAPIDoc, &doc); err != nil {
		t.Fatal(err)
	}
	enum := doc.Components.Schemas.Problem.Properties.Code.Enum
	if len(enum) == 0 {
		t.Fatal("the Problem schema has no code enum")
	}
	for code := range problemMeta {
		if !slices.Contains(enum, code) {
			t.Errorf("openapi.json does not list the problem code %q", code)
		}
	}
	for _, code := range enum {
		if _, ok := problemMeta[code]; !ok {
			t.Errorf("openapi.json lists %q, which the server never sends", code)
		}
	}
}
