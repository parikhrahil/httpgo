package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/parikhrahil/httpgo/internal/config"
)

// CollectionContext holds the env state for a single collection command run.
// Each env file (globalenv and <namespace>/env) is read from disk once at
// construction; mutations are applied to the in-memory maps and written back
// at most once per file via Persist.
type CollectionContext struct {
	Dir         string
	Namespace   string
	global      map[string]string
	local       map[string]string
	globalDirty bool
	localDirty  bool
}

func NewCollectionContext(dir, namespace string) (*CollectionContext, error) {
	global, err := loadEnvAllowMissing(config.GetGlobalEnvFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load global env: %w", err)
	}
	local, err := loadEnvAllowMissing(getFilePath(dir, namespace, "env"))
	if err != nil {
		return nil, fmt.Errorf("failed to load %s env: %w", namespace, err)
	}
	return &CollectionContext{
		Dir:       dir,
		Namespace: namespace,
		global:    global,
		local:     local,
	}, nil
}

func (c *CollectionContext) ClearLocal(keys []string) {
	if removeKeys(c.local, keys) {
		c.localDirty = true
	}
}

func (c *CollectionContext) ClearGlobal(keys []string) {
	if removeKeys(c.global, keys) {
		c.globalDirty = true
	}
}

func (c *CollectionContext) OverrideLocal(pairs []string) {
	if applyOverrides(c.local, pairs) {
		c.localDirty = true
	}
}

func (c *CollectionContext) OverrideGlobal(pairs []string) {
	if applyOverrides(c.global, pairs) {
		c.globalDirty = true
	}
}

// Persist writes each modified env file exactly once. Files that were not
// mutated are skipped, so a no-flag collection run touches zero env files.
func (c *CollectionContext) Persist() error {
	if c.globalDirty {
		if err := writeEnvFile(config.GetGlobalEnvFile(), c.global); err != nil {
			return fmt.Errorf("failed to persist global env: %w", err)
		}
		c.globalDirty = false
	}
	if c.localDirty {
		if err := writeEnvFile(getFilePath(c.Dir, c.Namespace, "env"), c.local); err != nil {
			return fmt.Errorf("failed to persist %s env: %w", c.Namespace, err)
		}
		c.localDirty = false
	}
	return nil
}

// Env returns the merged effective env: globalenv overridden by namespace env.
func (c *CollectionContext) Env() map[string]string {
	return merge(c.global, c.local)
}

// ParseNamedRequest reads <namespace>/http and substitutes {KEY} placeholders
// using the in-memory env. The env files are not re-read.
func (c *CollectionContext) ParseNamedRequest(name string) (*http.Request, error) {
	return parseNamedRequest(c.Dir, c.Namespace, name, c.Env())
}

func loadEnvAllowMissing(path string) (map[string]string, error) {
	env, err := config.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	return env, err
}

func removeKeys(m map[string]string, keys []string) bool {
	dirty := false
	for _, k := range keys {
		if _, ok := m[k]; ok {
			delete(m, k)
			dirty = true
		}
	}
	return dirty
}

func applyOverrides(m map[string]string, pairs []string) bool {
	if len(pairs) == 0 {
		return false
	}
	for k, v := range parseVariables(pairs) {
		m[k] = v
	}
	return true
}

func writeEnvFile(filePath string, variables map[string]string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", filePath, err)
	}
	defer file.Close()
	for k, v := range variables {
		if _, err := fmt.Fprintf(file, "%s=%s\n", k, v); err != nil {
			return fmt.Errorf("failed to write to %s: %w", filePath, err)
		}
	}
	return nil
}
