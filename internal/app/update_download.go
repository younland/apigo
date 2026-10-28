package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maxUpdateDownload     int64 = 512 << 20
	updateDownloadTimeout       = 15 * time.Minute
)

type updateProgress func(downloaded, total int64)

type releaseDownload struct {
	release  release
	dir      string
	goos     string
	goarch   string
	client   *http.Client
	progress updateProgress
}

func (r release) assetFor(goos, goarch string) (releaseAsset, error) {
	name, err := updateAssetName(r.Version, goos, goarch)
	if err != nil {
		return releaseAsset{}, err
	}
	for _, asset := range r.Assets {
		if asset.Name != name {
			continue
		}
		if err := validateUpdateAsset(asset); err != nil {
			return releaseAsset{}, err
		}
		return asset, nil
	}
	return releaseAsset{}, fmt.Errorf("新版本缺少当前平台的安装包 %s", name)
}

func updateAssetName(version, goos, goarch string) (string, error) {
	switch goos {
	case "windows":
		return fmt.Sprintf("apigo-%s-windows-%s-setup.exe", version, goarch), nil
	case "darwin":
		return fmt.Sprintf("apigo-%s-darwin-universal.dmg", version), nil
	case "linux":
		return fmt.Sprintf("apigo-%s-linux-%s.deb", version, goarch), nil
	default:
		return "", fmt.Errorf("当前平台 %s 不支持自动下载更新", goos)
	}
}

func validateUpdateAsset(asset releaseAsset) error {
	u, err := url.Parse(asset.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("更新包地址不是有效的 HTTPS 地址")
	}
	_, err = updateDigest(asset.Digest)
	return err
}

func updateDigest(digest string) ([]byte, error) {
	text, ok := strings.CutPrefix(digest, "sha256:")
	if !ok || len(text) != sha256.Size*2 {
		return nil, errors.New("更新包缺少有效的 SHA-256 摘要")
	}
	sum, err := hex.DecodeString(text)
	if err != nil {
		return nil, errors.New("更新包缺少有效的 SHA-256 摘要")
	}
	return sum, nil
}

func (r release) download(ctx context.Context, dir string, progress updateProgress) (string, error) {
	return (releaseDownload{
		release: r, dir: dir, goos: runtime.GOOS, goarch: runtime.GOARCH, progress: progress,
	}).run(ctx)
}

func (d releaseDownload) run(ctx context.Context) (string, error) {
	asset, err := d.release.assetFor(d.goos, d.goarch)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, updateDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "apigo")
	client := d.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.Request.URL.Scheme != "https" {
		return "", errors.New("更新包下载被重定向到非 HTTPS 地址")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载更新包失败：%s", resp.Status)
	}
	if resp.ContentLength > maxUpdateDownload {
		return "", errors.New("更新包超过 512 MB 上限")
	}
	return d.write(resp, asset)
}

func (d releaseDownload) write(resp *http.Response, asset releaseAsset) (path string, err error) {
	if err := os.MkdirAll(d.dir, 0o700); err != nil {
		return "", err
	}
	path = filepath.Join(d.dir, asset.Name)
	temp := path + ".download"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(temp)
		}
	}()
	if err := d.copy(file, resp, asset); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(temp, path); err != nil {
		return "", err
	}
	keep = true
	return path, nil
}

func (d releaseDownload) copy(dst io.Writer, resp *http.Response, asset releaseAsset) error {
	hash := sha256.New()
	writer := io.MultiWriter(dst, hash)
	if d.progress != nil {
		writer = &updateProgressWriter{writer: writer, progress: d.progress, total: resp.ContentLength}
	}
	written, err := io.Copy(writer, io.LimitReader(resp.Body, maxUpdateDownload+1))
	if err != nil {
		return err
	}
	if written > maxUpdateDownload {
		return errors.New("更新包超过 512 MB 上限")
	}
	expected, _ := updateDigest(asset.Digest)
	if !bytes.Equal(hash.Sum(nil), expected) {
		return errors.New("更新包 SHA-256 校验不一致")
	}
	return nil
}

type updateProgressWriter struct {
	writer     io.Writer
	progress   updateProgress
	total      int64
	downloaded int64
}

func (w *updateProgressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.downloaded += int64(n)
	w.progress(w.downloaded, w.total)
	return n, err
}
