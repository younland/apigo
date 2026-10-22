package app

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// baseURLOf takes the baseurl out of the variables an environment holds: the
// manager shows it as the prefix URL of its own, which no row may delete.
func baseURLOf(rows []savedRow) (string, []savedRow) {
	value := ""
	rest := make([]savedRow, 0, len(rows))
	for _, row := range rows {
		if row.Key == store.BaseURLVariable {
			if value == "" {
				value = row.Value
			}
			continue
		}
		rest = append(rest, row)
	}
	return value, rest
}

// withBaseURL puts the prefix URL back among the variables an environment is
// written with: it is one of them on disk, the first of the list.
func withBaseURL(value string, rows []savedRow) []savedRow {
	return append([]savedRow{{
		Enabled: true, Key: store.BaseURLVariable, Value: strings.TrimSpace(value),
	}}, rows...)
}

// prefixURLField edits the prefix URL of the environment: the address a request
// goes out to when its own address starts with a slash. It stands on its own
// rather than among the variable rows, so it reads as what it is and cannot be
// deleted by mistake.
func (a *app) prefixURLField(c *ui.Context, p design.Palette) {
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "前置URL（baseurl）").Bold().FontSize(design.FontSizeBody).TextColor(p.Text)
			ui.Text(c, "接口URL 以 / 开头时接在它后面").FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted)
		})
		ui.TextInput(c, &a.envBaseURL).Grow(1).MinWidth(0).
			Placeholder("http://localhost:8080").Label("前置URL")
	})
}
