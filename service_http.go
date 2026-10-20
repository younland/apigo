package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apigo/widgets"
)

// maxResponseBytes caps what the app reads of a response, so a large body
// cannot exhaust memory or stall a frame.
const maxResponseBytes = 10 << 20

// requestSpec is the request the panes describe, ready to be built.
type requestSpec struct {
	method  string
	rawURL  string
	params  []widgets.Row
	headers []widgets.Row
	body    string
}

// response is what a send left behind: the status, how long it took, and
// the body, or the error that stopped it.
type response struct {
	status     int
	statusText string
	duration   time.Duration
	size       int
	headers    http.Header
	body       string
	err        error
}

// buildRequest turns the panes into an HTTP request: the URL with enabled
// query parameters, the enabled headers, and the body for the methods that
// carry one.
func buildRequest(spec requestSpec) (*http.Request, error) {
	raw := strings.TrimSpace(spec.rawURL)
	if raw == "" {
		return nil, errors.New("请求 URL 为空")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("无法解析 URL %q", spec.rawURL)
	}
	query := u.Query()
	for _, param := range spec.params {
		if param.Enabled && strings.TrimSpace(param.Key) != "" {
			query.Set(param.Key, param.Value)
		}
	}
	u.RawQuery = query.Encode()

	body := ""
	if carriesBody(spec.method) {
		body = spec.body
	}
	var reader io.Reader
	if strings.TrimSpace(body) != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(spec.method, u.String(), reader)
	if err != nil {
		return nil, err
	}
	for _, header := range spec.headers {
		if header.Enabled && strings.TrimSpace(header.Key) != "" {
			req.Header.Set(header.Key, header.Value)
		}
	}
	if reader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// carriesBody reports whether a method sends the body pane's text.
func carriesBody(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD":
		return false
	default:
		return true
	}
}

// execute sends the request and reads the response, capped at
// maxResponseBytes.
func execute(ctx context.Context, req *http.Request) *response {
	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return &response{duration: time.Since(start), err: describe(err)}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	out := &response{
		status:     resp.StatusCode,
		statusText: http.StatusText(resp.StatusCode),
		duration:   time.Since(start),
		size:       len(data),
		headers:    resp.Header,
	}
	if err != nil {
		out.err = describe(err)
		return out
	}
	if len(data) > maxResponseBytes {
		out.body = string(data[:maxResponseBytes])
		out.err = fmt.Errorf("响应体超过 %d MB，已截断", maxResponseBytes>>20)
		return out
	}
	out.body = string(data)
	return out
}

// describe turns the usual network failures into what the response pane
// tells the reader.
func describe(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("请求超时")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("请求已取消")
	}
	return fmt.Errorf("请求失败：%w", err)
}
