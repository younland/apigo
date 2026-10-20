package main

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
type environment struct {
	ID     string     `yaml:"id"`
	Name   string     `yaml:"name"`
	Scope  string     `yaml:"-"`
	Active bool       `yaml:"active,omitempty"`
	Vars   []savedRow `yaml:"variables,omitempty"`
}

// envDir is the directory a scope keeps its environments in.
func (s *store) envDir(scope string) string {
	if scope == "" || scope == scopeGlobal {
		return s.dir
	}
	return filepath.Join(s.collectionDir(scope), envDirName)
}

// envPath is the file one environment is kept in.
func (s *store) envPath(env environment) string {
	return filepath.Join(s.envDir(env.Scope), safeFileName(env.Name)+".yaml")
}

// loadGlobalEnvironments reads the environments that apply to every
// collection.
func (s *store) loadGlobalEnvironments() error {
	var envs []environment
	if err := readYAMLIfExists(filepath.Join(s.dir, "environments.yaml"), &envs); err != nil {
		return err
	}
	for i := range envs {
		envs[i].Scope = scopeGlobal
		if envs[i].ID == "" {
			envs[i].ID = environmentID(scopeGlobal, envs[i].Name)
		}
	}
	s.environments = append(s.environments, envs...)
	return nil
}

// readEnvironments reads the environments of a collection, one file each.
func (s *store) readEnvironments(scope string) ([]environment, error) {
	dir := s.envDir(scope)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	envs := make([]environment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}
		var env environment
		if err := readYAML(filepath.Join(dir, entry.Name()), &env); err != nil {
			return nil, err
		}
		env.Scope = scope
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

// saveEnvironments writes every environment where its scope keeps it, and
// removes the files of the ones that are gone.
func (s *store) saveEnvironments() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	global := []environment{}
	scopes := map[string]bool{}
	for _, env := range s.environments {
		if env.Scope == "" || env.Scope == scopeGlobal {
			global = append(global, env)
			continue
		}
		if err := s.writeEnvironment(env); err != nil {
			return err
		}
		scopes[env.Scope] = true
	}
	if err := writeYAML(filepath.Join(s.dir, "environments.yaml"), global); err != nil {
		return err
	}
	for _, col := range s.collections {
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
func (s *store) writeEnvironment(env environment) error {
	return writeYAML(s.envPath(env), env)
}

// pruneEnvDir removes the files of environments a collection no longer
// keeps, as a rename or a deletion leaves behind.
func (s *store) pruneEnvDir(scope string) error {
	dir := s.envDir(scope)
	kept := map[string]bool{}
	for _, env := range s.environments {
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
func (s *store) activeEnvironment(scope string) *environment {
	for i := range s.environments {
		if s.environments[i].Scope == scope && s.environments[i].Active {
			return &s.environments[i]
		}
	}
	return nil
}

// environment returns the environment with the given id, or nil.
func (s *store) environment(id string) *environment {
	for i := range s.environments {
		if s.environments[i].ID == id {
			return &s.environments[i]
		}
	}
	return nil
}

// activateEnvironment makes one environment the active one of its scope,
// deactivating the others of that scope.
func (s *store) activateEnvironment(id string) error {
	env := s.environment(id)
	if env == nil {
		return fmt.Errorf("no environment %q", id)
	}
	scope := env.Scope
	for i := range s.environments {
		if s.environments[i].Scope == scope {
			s.environments[i].Active = s.environments[i].ID == id
		}
	}
	return s.saveEnvironments()
}

// environmentsFor returns the environments of a collection, in the order
// they were made.
func (s *store) environmentsFor(collectionID string) []environment {
	out := make([]environment, 0, len(s.environments))
	for _, env := range s.environments {
		if env.Scope == collectionID {
			out = append(out, env)
		}
	}
	return out
}

// defaultEnvironmentNames are the environments a new collection starts
// with, the first of them active.
var defaultEnvironmentNames = []string{"本地", "测试", "生产"}

// addDefaultEnvironments gives a new collection its environments, each
// carrying an empty BaseURL variable for the request line's leading URL.
func (s *store) addDefaultEnvironments(collectionID string) error {
	for i, name := range defaultEnvironmentNames {
		s.environments = append(s.environments, environment{
			ID: s.newEnvironmentID(), Name: name, Scope: collectionID, Active: i == 0,
			Vars: []savedRow{{Enabled: true, Key: "BaseURL"}},
		})
	}
	return s.saveEnvironments()
}

// addEnvironment adds an environment with an empty BaseURL to a collection.
func (s *store) addEnvironment(collectionID, name string) (environment, error) {
	env := environment{
		ID: s.newEnvironmentID(), Name: s.uniqueEnvironmentName(collectionID, name),
		Scope: collectionID, Vars: []savedRow{{Enabled: true, Key: "BaseURL"}},
	}
	s.environments = append(s.environments, env)
	return env, s.saveEnvironments()
}

// uniqueEnvironmentName numbers a name already taken in a collection.
func (s *store) uniqueEnvironmentName(collectionID, name string) string {
	taken := make([]string, 0, len(s.environments))
	for _, env := range s.environments {
		if env.Scope == collectionID {
			taken = append(taken, env.Name)
		}
	}
	return uniqueName(taken, name)
}

// newEnvironmentID numbers the environments, which keep their id across a
// rename so the editor keeps editing the same one.
func (s *store) newEnvironmentID() string {
	ids := make([]string, len(s.environments))
	for i, env := range s.environments {
		ids[i] = env.ID
	}
	return nextID("env", ids)
}

// deleteEnvironment removes an environment and writes the rest, leaving the
// scope with an active one when it had none.
func (s *store) deleteEnvironment(id string) error {
	scope := ""
	for i, env := range s.environments {
		if env.ID == id {
			scope = env.Scope
			s.environments = append(s.environments[:i], s.environments[i+1:]...)
			break
		}
	}
	if scope != "" && s.activeEnvironment(scope) == nil {
		for i := range s.environments {
			if s.environments[i].Scope == scope {
				s.environments[i].Active = true
				break
			}
		}
	}
	return s.saveEnvironments()
}

// renameEnvironment makes an environment's file follow its new name, and
// reports the name it ended up with.
func (s *store) renameEnvironment(id, name string) (string, error) {
	env := s.environment(id)
	if env == nil {
		return "", fmt.Errorf("no environment %q", id)
	}
	if strings.TrimSpace(name) == "" {
		return env.Name, nil
	}
	env.Name = uniqueName(s.otherEnvironmentNames(env), name)
	return env.Name, s.saveEnvironments()
}

// otherEnvironmentNames lists the names of the environments sharing a
// scope, leaving out the one being renamed.
func (s *store) otherEnvironmentNames(self *environment) []string {
	names := make([]string, 0, len(s.environments))
	for _, env := range s.environments {
		if env.Scope == self.Scope && env.ID != self.ID {
			names = append(names, env.Name)
		}
	}
	return names
}
