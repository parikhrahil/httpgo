package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// setupHTTPGoHome points HOME at a fresh tmp dir and pre-creates the
// ~/.httpgo/collections layout. Returns the collections dir so callers can
// build namespaces under it.
//
// IMPORTANT: cmd/root.go's init() runs before t.Setenv can take effect, so
// the real ~/.httpgo/collections may have been touched at test-process start.
// That's pre-existing harmless behavior — tests still operate exclusively
// inside the tmp home because config.GetWorkingDirectory() reads HOME on
// every call.
func setupHTTPGoHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := filepath.Join(home, ".httpgo", "collections")
	require.NoError(t, os.MkdirAll(wd, 0o755))
	return wd
}

// makeFile writes content to <dir>/<elems...>, creating parent dirs as needed.
func makeFile(t *testing.T, dir string, content string, elems ...string) string {
	t.Helper()
	parts := append([]string{dir}, elems...)
	full := filepath.Join(parts...)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	return full
}

// captureStdout swaps os.Stdout for a pipe while fn runs and returns the
// printed output. The handlers in this package print via fmt.Println rather
// than cmd.OutOrStdout(), so this is the only way to observe their output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	require.NoError(t, w.Close())
	return <-done
}

// parseEnvOutput parses lines of KEY=VALUE produced by printEnv. Map iteration
// in Go is unordered, so callers compare via maps, not byte strings.
func parseEnvOutput(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			out[k] = v
		}
	}
	return out
}

// newCollectionTestCmd mirrors the flag wiring in collectionCmd's init() so
// tests can drive executeHTTP and applyVarFlags directly without going
// through Cobra's argument parser (which would leak state between tests).
func newCollectionTestCmd() *cobra.Command {
	c := &cobra.Command{Use: "collection"}
	f := c.Flags()
	p := c.PersistentFlags()
	f.String("output", "", "")
	f.String("tee", "", "")
	p.Bool("dry-run", false, "")
	f.Bool("prettify", true, "")
	f.Bool("raw", false, "")
	f.Bool("include-headers", false, "")
	p.Duration("timeout", 0, "")
	p.StringArray("vars", nil, "")
	p.StringArray("global-vars", nil, "")
	p.StringArray("unset", nil, "")
	p.StringArray("global-unset", nil, "")
	return c
}
