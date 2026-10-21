package req

import (
	"regexp"

	"apigo/internal/widgets"
)

// variableRef matches {{ name }} as environments write variables.
var variableRef = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// ResolveVars replaces every {{name}} in text with the variables, and returns
// the names it could not resolve, each once, in order.
func ResolveVars(text string, vars map[string]string) (string, []string) {
	var missing []string
	out := variableRef.ReplaceAllStringFunc(text, func(m string) string {
		name := variableRef.FindStringSubmatch(m)[1]
		value, ok := vars[name]
		if !ok {
			missing = appendUnique(missing, name)
			return m
		}
		return value
	})
	return out, missing
}

// ResolveRows resolves the keys and values of key/value rows.
func ResolveRows(rows []widgets.Row, vars map[string]string) ([]widgets.Row, []string) {
	out := make([]widgets.Row, len(rows))
	var missing []string
	for i, row := range rows {
		var found []string
		out[i] = row
		out[i].Key, found = ResolveVars(row.Key, vars)
		missing = append(missing, found...)
		out[i].Value, found = ResolveVars(row.Value, vars)
		missing = append(missing, found...)
	}
	return out, Dedupe(missing)
}

// Dedupe keeps the first of each name, in order.
func Dedupe(names []string) []string {
	out := names[:0]
	for _, name := range names {
		out = appendUnique(out, name)
	}
	return out
}

func appendUnique(names []string, name string) []string {
	for _, have := range names {
		if have == name {
			return names
		}
	}
	return append(names, name)
}
