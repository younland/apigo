package req

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	res := Execute(context.Background(), req)
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
	if res := Execute(context.Background(), req); res.Err == nil {
		t.Fatal("no error for a closed server")
	}
}
