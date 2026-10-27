package req

import (
	"net/url"
	"regexp"
	"strings"
)

// pathRef matches one {name} of an address, the place a path parameter is
// filled in. The braces of the variable syntax are two, so a {{name}} is left
// to ResolveVars.
var pathRef = regexp.MustCompile(`\{([^{}]+)\}`)

// PathNames lists the {names} an address holds, in the order it holds them,
// each once.
func PathNames(rawURL string) []string {
	var names []string
	for _, match := range pathRef.FindAllStringSubmatch(rawURL, -1) {
		names = appendUnique(names, match[1])
	}
	return names
}

// FillPath fills the {names} of an address with the values of its path
// parameters, and names the ones no value fills. A value is read as a path
// segment, so a slash in it stays part of the value rather than of the address.
func FillPath(rawURL string, values map[string]string) (string, []string) {
	var missing []string
	out := pathRef.ReplaceAllStringFunc(rawURL, func(match string) string {
		name := match[1 : len(match)-1]
		value := strings.TrimSpace(values[name])
		if value == "" {
			missing = appendUnique(missing, name)
			return match
		}
		return url.PathEscape(value)
	})
	return out, missing
}
