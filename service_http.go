package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"apigo/widgets"
)

// maxResponseBytes caps what the app reads of a response, so a large body
// cannot exhaust memory or stall a frame.
const maxResponseBytes = 10 << 20

// formPart is one field of a multipart form: its text, or the file it uploads.
type formPart struct {
	key      string
	value    string
	filePath string // absolute, "" for a text field
	fileName string // the name the file had, which the request shows
}

// requestSpec is the request the panes describe, ready to be built.
type requestSpec struct {
	method  string
	rawURL  string
	params  []widgets.Row
	headers []widgets.Row
	// bodyType is the format of the body, and body its text or form its rows,
	// as that format edits it.
	bodyType string
	body     string
	form     []formPart
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

	var body io.Reader
	contentType := ""
	if carriesBody(spec.method) {
		var err error
		if body, contentType, err = spec.bodyReader(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(spec.method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for _, header := range spec.headers {
		if header.Enabled && strings.TrimSpace(header.Key) != "" {
			req.Header.Set(header.Key, header.Value)
		}
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

// bodyReader builds the body the format asks for, with the Content-Type it
// sends: the text of a code body, the encoded rows of a form, or nothing for a
// format that has none or needs a file.
func (s requestSpec) bodyReader() (io.Reader, string, error) {
	format := bodyFormatOf(s.bodyType)
	switch format.edits {
	case editsRows:
		if format.key == "form-data" {
			return s.multipartBody()
		}
		values := url.Values{}
		for _, part := range s.form {
			if strings.TrimSpace(part.key) != "" {
				values.Set(part.key, part.value)
			}
		}
		if len(values) == 0 {
			return nil, "", nil
		}
		return strings.NewReader(values.Encode()), format.mime, nil
	case editsText:
		if strings.TrimSpace(s.body) == "" {
			return nil, "", nil
		}
		return strings.NewReader(s.body), format.mime, nil
	default:
		// none and the formats that need a file send no body, and say nothing
		// of a type either.
		return nil, "", nil
	}
}

// multipartBody writes the fields of a form-data body: the text of a field, or
// the file it holds, under the name the file came with.
func (s requestSpec) multipartBody() (io.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, part := range s.form {
		if strings.TrimSpace(part.key) == "" {
			continue
		}
		if part.filePath == "" {
			if err := w.WriteField(part.key, part.value); err != nil {
				return nil, "", err
			}
			continue
		}
		if err := s.writeFilePart(w, part); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

// writeFilePart adds one file field to a multipart body.
func (s requestSpec) writeFilePart(w *multipart.Writer, part formPart) error {
	src, err := os.Open(part.filePath)
	if err != nil {
		return fmt.Errorf("上传的文件 %q：%w", part.fileName, err)
	}
	defer src.Close()
	dst, err := w.CreateFormFile(part.key, part.fileName)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
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
