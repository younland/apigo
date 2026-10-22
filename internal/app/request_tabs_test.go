package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRequestTabsNameThePanes checks the panes of a request read in the
// language of the interface.
func TestRequestTabsNameThePanes(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	for _, want := range []string{"参数", "请求头", "请求体"} {
		if _, ok := tt.Find(want); !ok {
			t.Errorf("texts %q, want the %q pane", tt.Texts(), want)
		}
	}
	for _, gone := range []string{"Params", "Headers", "Body"} {
		if _, ok := tt.Find(gone); ok {
			t.Errorf("%q is still in the request tabs, want its name in Chinese", gone)
		}
	}
}

// TestOpenRequestShowsTheFirstConfiguredPane checks a request opens on the pane
// that holds what it was configured with, the parameters when it has nothing.
func TestOpenRequestShowsTheFirstConfiguredPane(t *testing.T) {
	tests := []struct {
		name string
		req  savedRequest
		want int
	}{
		{"parameters", savedRequest{Params: []savedRow{{Enabled: true, Key: "limit", Value: "10"}}}, 0},
		{"headers", savedRequest{Headers: []savedRow{{Enabled: true, Key: "Accept", Value: "*/*"}}}, 1},
		{"body", savedRequest{BodyType: "json", Body: `{"ok":true}`}, 2},
		{"form body", savedRequest{BodyType: "urlencoded",
			Form: []savedRow{{Enabled: true, Key: "a", Value: "1"}}}, 2},
		{"nothing", savedRequest{}, 0},
		{"parameters win", savedRequest{
			Params:  []savedRow{{Enabled: true, Key: "limit", Value: "10"}},
			Headers: []savedRow{{Enabled: true, Key: "Accept", Value: "*/*"}},
			Body:    "{}",
		}, 0},
	}
	for _, test := range tests {
		a := testApp(t)
		test.req.Path, test.req.Name, test.req.Kind = "新接口", "新接口", kindREST
		test.req.Method = "GET"
		a.openRequest(a.active, test.req)
		if got := a.current().reqTab; got != test.want {
			t.Errorf("%s: opens on pane %d, want %d", test.name, got, test.want)
		}
	}
}
