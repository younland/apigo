package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// scopeGlobal is the scope of environments that apply to every collection.
const scopeGlobal = "global"

// environment is a set of variables that applies globally or to one
// collection. A collection keeps its own in <集合>/.env/<环境>.yaml; the
// global ones stay in one file of the data directory.
type Environment struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Scope  string `yaml:"-"`
	Active bool   `yaml:"active,omitempty"`
	Vars   []Row  `yaml:"variables,omitempty"`
}

// envDir is the directory a scope keeps its environments in.
func (s *Store) EnvDir(scope string) string {
	if scope == "" || scope == scopeGlobal {
		return s.Dir
	}
	return filepath.Join(s.CollectionDir(scope), envDirName)
}

// envPath is the file one environment is kept in.
func (s *Store) envPath(env Environment) string {
	return filepath.Join(s.EnvDir(env.Scope), safeFileName(env.Name)+".yaml")
}

// loadGlobalEnvironments reads the environments that apply to every
// collection.
func (s *Store) loadGlobalEnvironments() error {
	var envs []Environment
	if err := readYAMLIfExists(filepath.Join(s.Dir, "environments.yaml"), &envs); err != nil {
		return err
	}
	for i := range envs {
		envs[i].Scope = scopeGlobal
		if envs[i].ID == "" {
			envs[i].ID = environmentID(scopeGlobal, envs[i].Name)
		}
		migrateBaseURL(&envs[i])
	}
	s.Environments = append(s.Environments, envs...)
	return nil
}

// readEnvironments reads the environments of a collection, one file each.
func (s *Store) readEnvironments(scope string) ([]Environment, error) {
	dir := s.EnvDir(scope)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	envs := make([]Environment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}
		var env Environment
		if err := readYAML(filepath.Join(dir, entry.Name()), &env); err != nil {
			return nil, err
		}
		env.Scope = scope
		migrateBaseURL(&env)
		if env.Name == "" {
			env.Name = interfaceName(entry.Name())
		}
		if env.ID == "" {
			env.ID = environmentID(scope, env.Name)
		}
		envs = append(envs, env)
	}
	return envs, nil
}

// environmentID names an environment the files do not name.
func environmentID(scope, name string) string {
	return scope + "/" + name
}

// migrateBaseURL reads the baseurl variable of an environment kept by an older
// build, which wrote it in capitals: its value is the reader's, so the request
// line keeps showing it under the name the app uses now.
func migrateBaseURL(env *Environment) {
	for i := range env.Vars {
		if env.Vars[i].Key == legacyBaseURLVariable {
			env.Vars[i].Key = BaseURLVariable
		}
	}
}

// saveEnvironments writes every environment where its scope keeps it, and
// removes the files of the ones that are gone.
func (s *Store) SaveEnvironments() error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	global := []Environment{}
	scopes := map[string]bool{}
	for _, env := range s.Environments {
		if env.Scope == "" || env.Scope == scopeGlobal {
			global = append(global, env)
			continue
		}
		if err := s.writeEnvironment(env); err != nil {
			return err
		}
		scopes[env.Scope] = true
	}
	if err := WriteYAML(filepath.Join(s.Dir, "environments.yaml"), global); err != nil {
		return err
	}
	for _, col := range s.Collections {
		scopes[col.ID] = true
	}
	for scope := range scopes {
		if err := s.pruneEnvDir(scope); err != nil {
			return err
		}
	}
	return nil
}

// writeEnvironment writes one environment's file.
func (s *Store) writeEnvironment(env Environment) error {
	return WriteYAML(s.envPath(env), env)
}

// pruneEnvDir removes the files of environments a collection no longer
// keeps, as a rename or a deletion leaves behind.
func (s *Store) pruneEnvDir(scope string) error {
	dir := s.EnvDir(scope)
	kept := map[string]bool{}
	for _, env := range s.Environments {
		if env.Scope == scope {
			kept[safeFileName(env.Name)+".yaml"] = true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || kept[entry.Name()] || !isYAMLFile(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// activeEnvironment returns the active environment of a scope, if any.
func (s *Store) ActiveEnvironment(scope string) *Environment {
	for i := range s.Environments {
		if s.Environments[i].Scope == scope && s.Environments[i].Active {
			return &s.Environments[i]
		}
	}
	return nil
}

// environment returns the environment with the given id, or nil.
func (s *Store) Environment(id string) *Environment {
	for i := range s.Environments {
		if s.Environments[i].ID == id {
			return &s.Environments[i]
		}
	}
	return nil
}

// activateEnvironment makes one environment the active one of its scope,
// deactivating the others of that scope.
func (s *Store) ActivateEnvironment(id string) error {
	env := s.Environment(id)
	if env == nil {
		return fmt.Errorf("no environment %q", id)
	}
	scope := env.Scope
	for i := range s.Environments {
		if s.Environments[i].Scope == scope {
			s.Environments[i].Active = s.Environments[i].ID == id
		}
	}
	return s.SaveEnvironments()
}

// environmentsFor returns the environments of a collection, in the order
// they were made.
func (s *Store) EnvironmentsFor(collectionID string) []Environment {
	out := make([]Environment, 0, len(s.Environments))
	for _, env := range s.Environments {
		if env.Scope == collectionID {
			out = append(out, env)
		}
	}
	return out
}

// BaseURLVariable is the variable a request that starts with a slash is read
// against: its value is the address the rest of the path follows, and the
// request line shows it before the field.
const BaseURLVariable = "baseurl"

// legacyBaseURLVariable is how the variable was written before it took its
// lowercase name: a collection kept by an older build reads as the current one.
const legacyBaseURLVariable = "BaseURL"

// defaultEnvironmentNames are the environments a new collection starts
// with, the first of them active.
var defaultEnvironmentNames = []string{"本地", "测试", "生产"}

// addDefaultEnvironments gives a new collection its environments, each
// carrying an empty baseurl variable for the request line's leading URL.
func (s *Store) addDefaultEnvironments(collectionID string) error {
	for i, name := range defaultEnvironmentNames {
		s.Environments = append(s.Environments, Environment{
			ID: s.newEnvironmentID(), Name: name, Scope: collectionID, Active: i == 0,
			Vars: []Row{{Enabled: true, Key: BaseURLVariable}},
		})
	}
	return s.SaveEnvironments()
}

// addEnvironment adds an environment with an empty baseurl to a collection.
func (s *Store) addEnvironment(collectionID, name string) (Environment, error) {
	env := Environment{
		ID: s.newEnvironmentID(), Name: s.uniqueEnvironmentName(collectionID, name),
		Scope: collectionID, Vars: []Row{{Enabled: true, Key: BaseURLVariable}},
	}
	s.Environments = append(s.Environments, env)
	return env, s.SaveEnvironments()
}

// uniqueEnvironmentName numbers a name already taken in a collection.
func (s *Store) uniqueEnvironmentName(collectionID, name string) string {
	taken := make([]string, 0, len(s.Environments))
	for _, env := range s.Environments {
		if env.Scope == collectionID {
			taken = append(taken, env.Name)
		}
	}
	return uniqueName(taken, name)
}

// newEnvironmentID numbers the environments, which keep their id across a
// rename so the editor keeps editing the same one.
func (s *Store) newEnvironmentID() string {
	ids := make([]string, len(s.Environments))
	for i, env := range s.Environments {
		ids[i] = env.ID
	}
	return nextID("env", ids)
}

// deleteEnvironment removes an environment and writes the rest, leaving the
// scope with an active one when it had none.
func (s *Store) deleteEnvironment(id string) error {
	scope := ""
	for i, env := range s.Environments {
		if env.ID == id {
			scope = env.Scope
			s.Environments = append(s.Environments[:i], s.Environments[i+1:]...)
			break
		}
	}
	if scope != "" && s.ActiveEnvironment(scope) == nil {
		for i := range s.Environments {
			if s.Environments[i].Scope == scope {
				s.Environments[i].Active = true
				break
			}
		}
	}
	return s.SaveEnvironments()
}

// renameEnvironment makes an environment's file follow its new name, and
// reports the name it ended up with.
func (s *Store) renameEnvironment(id, name string) (string, error) {
	env := s.Environment(id)
	if env == nil {
		return "", fmt.Errorf("no environment %q", id)
	}
	if strings.TrimSpace(name) == "" {
		return env.Name, nil
	}
	env.Name = uniqueName(s.otherEnvironmentNames(env), name)
	return env.Name, s.SaveEnvironments()
}

// otherEnvironmentNames lists the names of the environments sharing a
// scope, leaving out the one being renamed.
func (s *Store) otherEnvironmentNames(self *Environment) []string {
	names := make([]string, 0, len(s.Environments))
	for _, env := range s.Environments {
		if env.Scope == self.Scope && env.ID != self.ID {
			names = append(names, env.Name)
		}
	}
	return names
}

// cloneEnvironments copies the environments as they stand: what a dialog that
// is cancelled puts back.
func (s *Store) CloneEnvironments() []Environment {
	out := make([]Environment, len(s.Environments))
	for i, env := range s.Environments {
		out[i] = env
		out[i].Vars = append([]Row(nil), env.Vars...)
	}
	return out
}

// restoreEnvironments puts a copy back, undoing what a dialog changed.
func (s *Store) RestoreEnvironments(kept []Environment) { s.Environments = kept }

// selectEnvironment makes one environment the active one of its scope without
// writing it: the environment manager keeps the change until it is saved.
func (s *Store) SelectEnvironment(id string) error {
	env := s.Environment(id)
	if env == nil {
		return fmt.Errorf("no environment %q", id)
	}
	for i := range s.Environments {
		if s.Environments[i].Scope == env.Scope {
			s.Environments[i].Active = s.Environments[i].ID == id
		}
	}
	return nil
}

// nameEnvironment gives an environment a new name without writing it, and
// returns the name it ended up with.
func (s *Store) NameEnvironment(id, name string) string {
	env := s.Environment(id)
	if env == nil {
		return ""
	}
	if strings.TrimSpace(name) == "" {
		return env.Name
	}
	env.Name = uniqueName(s.otherEnvironmentNames(env), name)
	return env.Name
}

// newEnvironment makes an environment with an empty baseurl without writing
// it, as the manager adds one until it is saved.
func (s *Store) NewEnvironment(collectionID, name string) Environment {
	env := Environment{
		ID: s.newEnvironmentID(), Name: s.uniqueEnvironmentName(collectionID, name),
		Scope: collectionID, Vars: []Row{{Enabled: true, Key: BaseURLVariable}},
	}
	s.Environments = append(s.Environments, env)
	return env
}

// forgetEnvironment takes an environment out of memory without writing it.
func (s *Store) ForgetEnvironment(id string) {
	for i, env := range s.Environments {
		if env.ID != id {
			continue
		}
		scope := env.Scope
		s.Environments = append(s.Environments[:i], s.Environments[i+1:]...)
		if s.ActiveEnvironment(scope) == nil {
			for j := range s.Environments {
				if s.Environments[j].Scope == scope {
					s.Environments[j].Active = true
					break
				}
			}
		}
		return
	}
}
