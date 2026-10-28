package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testReleaseAssets(version string) []releaseAsset {
	return []releaseAsset{
		{Name: "apigo-" + version + "-windows-amd64-setup.exe", URL: "https://example.com/windows", Digest: "sha256:" + strings.Repeat("a", 64)},
		{Name: "apigo-" + version + "-darwin-universal.dmg", URL: "https://example.com/macos", Digest: "sha256:" + strings.Repeat("b", 64)},
		{Name: "apigo-" + version + "-linux-amd64.deb", URL: "https://example.com/amd64", Digest: "sha256:" + strings.Repeat("c", 64)},
		{Name: "apigo-" + version + "-linux-arm64.deb", URL: "https://example.com/arm64", Digest: "sha256:" + strings.Repeat("d", 64)},
	}
}

func TestReleaseAssetForPlatforms(t *testing.T) {
	rel := release{Version: "0.2.0", Assets: testReleaseAssets("0.2.0")}
	for _, test := range []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "apigo-0.2.0-windows-amd64-setup.exe"},
		{"darwin", "arm64", "apigo-0.2.0-darwin-universal.dmg"},
		{"linux", "amd64", "apigo-0.2.0-linux-amd64.deb"},
		{"linux", "arm64", "apigo-0.2.0-linux-arm64.deb"},
	} {
		asset, err := rel.assetFor(test.goos, test.goarch)
		if err != nil {
			t.Fatalf("%s/%s: %v", test.goos, test.goarch, err)
		}
		if asset.Name != test.want {
			t.Errorf("%s/%s asset %q, want %q", test.goos, test.goarch, asset.Name, test.want)
		}
	}
}

func TestReleaseAssetRejectsMissingOrUnsafeMetadata(t *testing.T) {
	for _, test := range []struct {
		name  string
		asset releaseAsset
	}{
		{"missing digest", releaseAsset{Name: "apigo-0.2.0-windows-amd64-setup.exe", URL: "https://example.com/setup"}},
		{"plain HTTP", releaseAsset{Name: "apigo-0.2.0-windows-amd64-setup.exe", URL: "http://example.com/setup", Digest: "sha256:abc"}},
		{"bad digest", releaseAsset{Name: "apigo-0.2.0-windows-amd64-setup.exe", URL: "https://example.com/setup", Digest: "sha256:xyz"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rel := release{Version: "0.2.0", Assets: []releaseAsset{test.asset}}
			if _, err := rel.assetFor("windows", "amd64"); err == nil {
				t.Fatalf("assetFor accepted unsafe metadata %+v", test.asset)
			}
		})
	}
	rel := release{Version: "0.2.0"}
	if _, err := rel.assetFor("windows", "amd64"); err == nil {
		t.Error("a release without its platform asset was accepted")
	}
}

func TestReleaseDownloadVerifiesSHA256(t *testing.T) {
	payload := []byte("verified installer bytes")
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(payload))
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	rel := release{Version: "0.2.0", Assets: []releaseAsset{{
		Name: "apigo-0.2.0-windows-amd64-setup.exe", URL: server.URL, Digest: digest,
	}}}
	dir := t.TempDir()
	path, err := (releaseDownload{
		release: rel, dir: dir, goos: "windows", goarch: "amd64", client: server.Client(),
	}).run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("downloaded %q, want %q", got, payload)
	}
}

func TestReleaseDownloadRejectsBadChecksum(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}))
	defer server.Close()
	rel := release{Version: "0.2.0", Assets: []releaseAsset{{
		Name: "apigo-0.2.0-windows-amd64-setup.exe", URL: server.URL,
		Digest: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("expected"))),
	}}}
	dir := t.TempDir()
	_, err := (releaseDownload{
		release: rel, dir: dir, goos: "windows", goarch: "amd64", client: server.Client(),
	}).run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("checksum error %v, want SHA-256 mismatch", err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("failed download left %v in %s", names, filepath.Clean(dir))
	}
}

func TestUpdateDownloadsAndStartsInstaller(t *testing.T) {
	a, tt := updateApp(t, release{Version: "0.2.0"}, nil)
	opened := ""
	quit := false
	a.fetchUpdate = func(ctx context.Context, rel release, progress updateProgress) (string, error) {
		progress(5, 10)
		return `C:\Temp\apigo-setup.exe`, nil
	}
	a.opener = func(path string) error {
		opened = path
		return nil
	}
	a.quit = func() { quit = true }
	a.checkUpdates(true)
	tt.Frame()
	if !tt.HasText("下载并更新") {
		t.Fatalf("texts %q, want the download action", tt.Texts())
	}
	if err := tt.Click("确认下载并更新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if opened != `C:\Temp\apigo-setup.exe` {
		t.Errorf("opened %q, want the verified installer", opened)
	}
	if !quit {
		t.Error("the app did not quit for the installer")
	}
	if got := a.updateNote; !strings.Contains(got, "正在启动安装程序") {
		t.Errorf("note %q, want the installer status", got)
	}
}

func TestUpdateDownloadFailureDoesNotInstall(t *testing.T) {
	a, tt := updateApp(t, release{Version: "0.2.0"}, nil)
	opened := false
	quit := false
	a.fetchUpdate = func(context.Context, release, updateProgress) (string, error) {
		return "", fmt.Errorf("SHA-256 校验不一致")
	}
	a.opener = func(string) error { opened = true; return nil }
	a.quit = func() { quit = true }
	a.checkUpdates(true)
	tt.Frame()
	if err := tt.Click("确认下载并更新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if opened || quit {
		t.Errorf("failed download opened=%v quit=%v, want neither", opened, quit)
	}
	if got := a.updateNote; !strings.Contains(got, "SHA-256 校验不一致") {
		t.Errorf("note %q, want the checksum failure", got)
	}
	if a.downloadingUpdate {
		t.Error("the failed download remained in progress")
	}
}

func TestUpdateDoesNotInstallAfterASaveFailure(t *testing.T) {
	a, tt := updateApp(t, release{Version: "0.2.0"}, nil)
	a.store.Collections[0].Root.Requests = nil
	opened := false
	quit := false
	a.fetchUpdate = func(context.Context, release, updateProgress) (string, error) {
		return `C:\Temp\apigo-setup.exe`, nil
	}
	a.opener = func(string) error { opened = true; return nil }
	a.quit = func() { quit = true }
	a.checkUpdates(true)
	tt.Frame()
	if err := tt.Click("确认下载并更新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if opened || quit {
		t.Errorf("failed save opened=%v quit=%v, want neither", opened, quit)
	}
	if got := a.updateNote; !strings.Contains(got, "接口保存失败") {
		t.Errorf("note %q, want the save failure", got)
	}
}

func TestUpdateDownloadCannotStartTwice(t *testing.T) {
	a, _ := updateApp(t, release{Version: "0.2.0"}, nil)
	calls := 0
	a.fetchUpdate = func(context.Context, release, updateProgress) (string, error) {
		calls++
		return "", nil
	}
	a.downloadingUpdate = true
	a.startUpdateDownload(release{Version: "0.2.0"})
	if calls != 0 {
		t.Errorf("%d downloads started while one was active, want none", calls)
	}
}
