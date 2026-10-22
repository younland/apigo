package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// fetchLimit caps how much of a document is read from a URL: a document is a
// description of interfaces, not a body to hold in memory.
const fetchLimit = 32 << 20

// fetchTimeout is how long a document is waited for before the import says it
// could not be read.
const fetchTimeout = 30 * time.Second

// Fetch reads the document a URL serves, JSON or YAML.
func Fetch(ctx context.Context, url string) ([]byte, error) {
	address := strings.TrimSpace(url)
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return nil, errors.New("URL 要以 http:// 或 https:// 开头")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")
	response, err := (&http.Client{Timeout: fetchTimeout}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("读取 %s 失败：%s", address, response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, fetchLimit))
	if err != nil {
		return nil, err
	}
	return data, nil
}
