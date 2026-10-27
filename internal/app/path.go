package app

import (
	"apigo/internal/req"
	"apigo/internal/widgets"
)

// syncPathParams makes the path table the names the address takes: an address of
// /pets/{id} holds one row, named id, whose value the reader fills in.
func (t *tab) syncPathParams() {
	t.pathParams.Fixed = true
	names := req.PathNames(t.url)
	if sameParamNames(t.pathParams.Rows, names) {
		return
	}
	rows := make([]widgets.Param, 0, len(names))
	for _, name := range names {
		row := widgets.Param{Enabled: true, Key: name}
		for _, have := range t.pathParams.Rows {
			if have.Key == name {
				row = have
				break
			}
		}
		rows = append(rows, row)
	}
	t.pathParams.Rows = rows
}

// sameParamNames reports whether a path table already holds exactly the names an
// address takes, in the order it takes them.
func sameParamNames(rows []widgets.Param, names []string) bool {
	if len(rows) != len(names) {
		return false
	}
	for i, name := range names {
		if rows[i].Key != name {
			return false
		}
	}
	return true
}

// pathValues are the values the {names} of an address are sent with, the
// variables of each resolved, and the names no value fills.
func (a *app) pathValues(t *tab, vars map[string]string) (map[string]string, []string) {
	values := make(map[string]string, len(t.pathParams.Rows))
	var missing []string
	for _, row := range t.pathParams.Enabled() {
		value, found := req.ResolveVars(row.Value, vars)
		missing = append(missing, found...)
		values[row.Key] = value
	}
	return values, missing
}
