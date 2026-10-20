package main

import (
	"regexp"

	"apigo/widgets"
)

// variableRef matches {{ name }} as environments write variables.
var variableRef = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// resolveVars replaces every {{name}} in text with the variables, and
// returns the names it could not resolve, each once, in order.
func resolveVars(text string, vars map[string]string) (string, []string) {
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

// resolveRows resolves the keys and values of key/value rows.
func resolveRows(rows []widgets.Row, vars map[string]string) ([]widgets.Row, []string) {
	out := make([]widgets.Row, len(rows))
	var missing []string
	for i, row := range rows {
		var found []string
		out[i] = row
		out[i].Key, found = resolveVars(row.Key, vars)
		missing = append(missing, found...)
		out[i].Value, found = resolveVars(row.Value, vars)
		missing = append(missing, found...)
	}
	return out, dedupe(missing)
}

// varsFor merges the active environments that apply to a collection: the
// global one first, the collection's after it, so the collection overrides
// a global variable of the same name.
func (s *store) varsFor(collectionID string) map[string]string {
	vars := map[string]string{}
	for _, scope := range []string{scopeGlobal, collectionID} {
		if scope == "" {
			continue
		}
		env := s.activeEnvironment(scope)
		if env == nil {
			continue
		}
		for _, row := range env.Vars {
			if row.Enabled && row.Key != "" {
				vars[row.Key] = row.Value
			}
		}
	}
	return vars
}

func appendUnique(names []string, name string) []string {
	for _, have := range names {
		if have == name {
			return names
		}
	}
	return append(names, name)
}

func dedupe(names []string) []string {
	out := names[:0]
	for _, name := range names {
		out = appendUnique(out, name)
	}
	return out
}
