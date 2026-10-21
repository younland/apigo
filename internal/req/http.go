package req

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

	"apigo/internal/widgets"
)

// maxResponseBytes caps what the app reads of a response, so a large body
// cannot exhaust memory or stall a frame.
const maxResponseBytes = 10 << 20

// Part is one field of a multipart form: its text, or the file it uploads.
type Part struct {
	Key      string
	Value    string
	FilePath string // absolute, "" for a text field
	FileName string // the name the file had, which the request shows
}

// Spec is the request the panes describe, ready to be built.
type Spec struct {
	Method  string
	RawURL  string
	Params  []widgets.Row
	Headers []widgets.Row
	// BodyType is the format of the body, and Body its text or Form its rows,
	// as that format edits it.
	BodyType string
	Body     string
	Form     []Part
}

// Response is what a send left behind: the status, how long it took, and the
// body, or the error that stopped it.
type Response struct {
	Status     int
	StatusText string
	Duration   time.Duration
	Size       int
	Headers    http.Header
	Body       string
	Err        error
}

// buildRequest turns the panes into an HTTP request: the URL with enabled
// query parameters, the enabled headers, and the body for the methods that
// carry one.
func BuildRequest(spec Spec) (*http.Request, error) {
	raw := strings.TrimSpace(spec.RawURL)
	if raw == "" {
		return nil, errors.New("请求 URL 为空")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("无法解析 URL %q", spec.RawURL)
	}
	query := u.Query()
	for _, param := range spec.Params {
		if param.Enabled && strings.TrimSpace(param.Key) != "" {
			query.Set(param.Key, param.Value)
		}
	}
	u.RawQuery = query.Encode()

	var body io.Reader
	contentType := ""
	if carriesBody(spec.Method) {
		var err error
		if body, contentType, err = spec.bodyReader(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(spec.Method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for _, header := range spec.Headers {
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
func (s Spec) bodyReader() (io.Reader, string, error) {
	format := FormatOf(s.BodyType)
	switch format.Edits {
	case EditsRows:
		if format.Key == "form-data" {
			return s.multipartBody()
		}
		values := url.Values{}
		for _, part := range s.Form {
			if strings.TrimSpace(part.Key) != "" {
				values.Set(part.Key, part.Value)
			}
		}
		if len(values) == 0 {
			return nil, "", nil
		}
		return strings.NewReader(values.Encode()), format.MIME, nil
	case EditsText:
		if strings.TrimSpace(s.Body) == "" {
			return nil, "", nil
		}
		return strings.NewReader(s.Body), format.MIME, nil
	default:
		// none and the formats that need a file send no body, and say nothing
		// of a type either.
		return nil, "", nil
	}
}

// multipartBody writes the fields of a form-data Body: the text of a field, or
// the file it holds, under the name the file came with.
func (s Spec) multipartBody() (io.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, part := range s.Form {
		if strings.TrimSpace(part.Key) == "" {
			continue
		}
		if part.FilePath == "" {
			if err := w.WriteField(part.Key, part.Value); err != nil {
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
func (s Spec) writeFilePart(w *multipart.Writer, part Part) error {
	src, err := os.Open(part.FilePath)
	if err != nil {
		return fmt.Errorf("上传的文件 %q：%w", part.FileName, err)
	}
	defer src.Close()
	dst, err := w.CreateFormFile(part.Key, part.FileName)
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
func Execute(ctx context.Context, req *http.Request) *Response {
	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return &Response{Duration: time.Since(start), Err: describe(err)}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	out := &Response{
		Status:     resp.StatusCode,
		StatusText: http.StatusText(resp.StatusCode),
		Duration:   time.Since(start),
		Size:       len(data),
		Headers:    resp.Header,
	}
	if err != nil {
		out.Err = describe(err)
		return out
	}
	if len(data) > maxResponseBytes {
		out.Body = string(data[:maxResponseBytes])
		out.Err = fmt.Errorf("响应体超过 %d MB，已截断", maxResponseBytes>>20)
		return out
	}
	out.Body = string(data)
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
