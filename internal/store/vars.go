package store

// VarsFor merges the active environments that apply to a collection: the
// global one first, the collection's after it, so the collection overrides
// a global variable of the same name.
func (s *Store) VarsFor(collectionID string) map[string]string {
	vars := map[string]string{}
	for _, scope := range []string{scopeGlobal, collectionID} {
		if scope == "" {
			continue
		}
		env := s.ActiveEnvironment(scope)
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
