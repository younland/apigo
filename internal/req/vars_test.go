package req

import "testing"

// TestResolveVars checks a variable the environment does not hold is kept as
// written and reported once, beside the ones it resolves.
func TestResolveVars(t *testing.T) {
	out, missing := ResolveVars("https://{{host}}/{{ path }}?x={{host}}", map[string]string{"host": "api.example.com"})
	if out != "https://api.example.com/{{ path }}?x=api.example.com" {
		t.Errorf("resolved %q, want the host replaced", out)
	}
	if len(missing) != 1 || missing[0] != "path" {
		t.Errorf("missing %+v, want only path", missing)
	}
}
