package app

import (
	"strings"

	"apigo/internal/req"
	"apigo/internal/store"
)

// targetURL is the address a request goes out to: an address that starts with a
// slash is read against the baseurl of the environment in use, so a collection
// switches hosts without touching its interfaces. The missing names are the
// variables the address still holds.
func (a *app) targetURL(t *tab) (string, []string) {
	vars := a.store.VarsFor(a.active)
	raw := strings.TrimSpace(t.url)
	if strings.HasPrefix(raw, "/") {
		base := strings.TrimRight(strings.TrimSpace(vars[store.BaseURLVariable]), "/")
		if base == "" {
			return "", []string{store.BaseURLVariable}
		}
		if !strings.Contains(base, "://") {
			base = schemeOf(t.kind) + "://" + base
		}
		raw = base + raw
	}
	return req.ResolveVars(raw, vars)
}

// schemeOf is the scheme an address without one is read under: a socket speaks
// ws, everything else http.
func schemeOf(k kind) string {
	if k == kindWebSocket || k == kindSocketIO {
		return "ws"
	}
	return "http"
}

// baseURLPrefix is the prefix URL the address hangs from: the baseurl of the
// environment in use, and whether the address is a path under it. An address
// that is not a path, and an environment that holds no baseurl yet, read as no
// prefix at all.
func (a *app) baseURLPrefix(t *tab) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(t.url), "/") {
		return "", false
	}
	base := strings.TrimRight(strings.TrimSpace(a.store.VarsFor(a.active)[store.BaseURLVariable]), "/")
	return base, base != ""
}

// missingText says what an address could not be read for: the variables it
// still holds, or the prefix URL that a path has none of.
func missingText(missing []string) string {
	for _, name := range missing {
		if name == store.BaseURLVariable {
			return "URL 以 / 开头，但当前环境没有设置前置URL"
		}
	}
	return "未解析的环境变量：" + strings.Join(missing, "、")
}
