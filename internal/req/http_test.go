package req

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"apigo/internal/widgets"
)

func TestBuildRequestMergesParamsAndHeaders(t *testing.T) {
	req, err := BuildRequest(Spec{
		Method: "POST",
		RawURL: "http://example.com/pets?page=1",
		Params: []widgets.Row{
			{Enabled: true, Key: "limit", Value: "10"},
			{Enabled: false, Key: "skip", Value: "1"},
			{Enabled: true, Key: ""},
		},
		Headers: []widgets.Row{
			{Enabled: true, Key: "Accept", Value: "application/json"},
			{Enabled: false, Key: "X-Off", Value: "1"},
		},
		Body: `{"name":"Ada"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := req.URL.String(), "http://example.com/pets?limit=10&page=1"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
	if got := req.Header.Get("X-Off"); got != "" {
		t.Errorf("X-Off = %q, want the disabled header left out", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json for a JSON body", got)
	}
}

func TestBuildRequestAddsScheme(t *testing.T) {
	req, err := BuildRequest(Spec{Method: "GET", RawURL: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := req.URL.String(), "https://example.com"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestBuildRequestRejectsEmptyURL(t *testing.T) {
	if _, err := BuildRequest(Spec{Method: "GET", RawURL: "  "}); err == nil {
		t.Fatal("no error for an empty URL")
	}
}

func TestBuildRequestDropsBodyForGet(t *testing.T) {
	req, err := BuildRequest(Spec{Method: "GET", RawURL: "http://example.com", Body: `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if req.Body != nil {
		t.Error("GET carried a body")
	}
}

func TestExecuteReadsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	req, err := BuildRequest(Spec{Method: "GET", RawURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := Execute(context.Background(), req, 0)
	if res.Err != nil {
		t.Fatalf("execute: %v", res.Err)
	}
	if res.Status != http.StatusCreated || res.StatusText != "Created" {
		t.Errorf("status %d %q, want 201 Created", res.Status, res.StatusText)
	}
	if res.Body != `{"ok":true}` || res.Size != len(res.Body) {
		t.Errorf("body %q size %d, want the JSON body and its size", res.Body, res.Size)
	}
	if res.Headers.Get("X-Test") != "yes" {
		t.Errorf("headers %v, want X-Test", res.Headers)
	}
	if res.Duration > time.Second {
		t.Errorf("duration %v, want a local response under a second", res.Duration)
	}
}

func TestExecuteReportsNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	req, err := BuildRequest(Spec{Method: "GET", RawURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if res := Execute(context.Background(), req, 0); res.Err == nil {
		t.Fatal("no error for a closed server")
	}
}

// TestExecuteCapsTheBody checks a body past the limit comes back cut to it,
// with what the limit was said in the error.
func TestExecuteCapsTheBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 3000))
	}))
	defer server.Close()

	req, err := BuildRequest(Spec{Method: "GET", RawURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	const limit = 2048
	res := Execute(context.Background(), req, limit)
	if len(res.Raw) != limit || res.Size != limit {
		t.Errorf("body %d bytes, size %d, want the %d of the limit", len(res.Raw), res.Size, limit)
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "2 KB") {
		t.Errorf("error %v, want one naming the 2 KB limit", res.Err)
	}
}

// TestExecuteDefaultsItsLimit checks a limit of zero takes the default, so a
// caller that says nothing still gets a body.
func TestExecuteDefaultsItsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("small"))
	}))
	defer server.Close()

	req, err := BuildRequest(Spec{Method: "GET", RawURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if res := Execute(context.Background(), req, 0); res.Err != nil || res.Body != "small" {
		t.Errorf("body %q error %v, want the body read with the default limit", res.Body, res.Err)
	}
}

// TestLimitTextWritesSizes checks a cap reads as a person expects it.
func TestLimitTextWritesSizes(t *testing.T) {
	tests := []struct {
		limit int64
		want  string
	}{
		{2 << 10, "2 KB"},
		{1 << 20, "1 MB"},
		{100 << 20, "100 MB"},
		{1<<20 + 1<<19, "1.5 MB"},
	}
	for _, test := range tests {
		if got := limitText(test.limit); got != test.want {
			t.Errorf("limitText(%d) = %q, want %q", test.limit, got, test.want)
		}
	}
}
