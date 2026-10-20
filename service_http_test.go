package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"apigo/widgets"
)

func TestBuildRequestMergesParamsAndHeaders(t *testing.T) {
	req, err := buildRequest(requestSpec{
		method: "POST",
		rawURL: "http://example.com/pets?page=1",
		params: []widgets.Row{
			{Enabled: true, Key: "limit", Value: "10"},
			{Enabled: false, Key: "skip", Value: "1"},
			{Enabled: true, Key: ""},
		},
		headers: []widgets.Row{
			{Enabled: true, Key: "Accept", Value: "application/json"},
			{Enabled: false, Key: "X-Off", Value: "1"},
		},
		body: `{"name":"Ada"}`,
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
	req, err := buildRequest(requestSpec{method: "GET", rawURL: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := req.URL.String(), "https://example.com"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestBuildRequestRejectsEmptyURL(t *testing.T) {
	if _, err := buildRequest(requestSpec{method: "GET", rawURL: "  "}); err == nil {
		t.Fatal("no error for an empty URL")
	}
}

func TestBuildRequestDropsBodyForGet(t *testing.T) {
	req, err := buildRequest(requestSpec{method: "GET", rawURL: "http://example.com", body: `{"a":1}`})
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

	req, err := buildRequest(requestSpec{method: "GET", rawURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := execute(context.Background(), req)
	if res.err != nil {
		t.Fatalf("execute: %v", res.err)
	}
	if res.status != http.StatusCreated || res.statusText != "Created" {
		t.Errorf("status %d %q, want 201 Created", res.status, res.statusText)
	}
	if res.body != `{"ok":true}` || res.size != len(res.body) {
		t.Errorf("body %q size %d, want the JSON body and its size", res.body, res.size)
	}
	if res.headers.Get("X-Test") != "yes" {
		t.Errorf("headers %v, want X-Test", res.headers)
	}
	if res.duration > time.Second {
		t.Errorf("duration %v, want a local response under a second", res.duration)
	}
}

func TestExecuteReportsNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	req, err := buildRequest(requestSpec{method: "GET", rawURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if res := execute(context.Background(), req); res.err == nil {
		t.Fatal("no error for a closed server")
	}
}
