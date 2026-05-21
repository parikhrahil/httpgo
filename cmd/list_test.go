package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newListCmdWithAllFlag mirrors listCmd's flag wiring so listRun can be driven
// directly. The shared listCmd carries state across tests; a fresh cmd avoids
// that bleed.
func newListCmdWithAllFlag(all bool) *cobra.Command {
	c := &cobra.Command{Use: "list"}
	c.Flags().BoolP("all", "a", false, "")
	if all {
		_ = c.Flags().Set("all", "true")
	}
	return c
}

func TestListRun_MissingWorkingDirErrors(t *testing.T) {
	// HOME points at a tmp dir but ~/.httpgo/collections doesn't exist.
	t.Setenv("HOME", t.TempDir())
	err := listRun(newListCmdWithAllFlag(false), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not fetch available namespaces")
}

func TestListRun_EmptyWorkingDirProducesNoOutput(t *testing.T) {
	setupHTTPGoHome(t)
	out := captureStdout(t, func() {
		require.NoError(t, listRun(newListCmdWithAllFlag(false), nil))
	})
	assert.Empty(t, strings.TrimSpace(out))
}

func TestListRun_NoFlagsListsNamespaces(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "alpha"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "beta"), 0o755))

	out := captureStdout(t, func() {
		require.NoError(t, listRun(newListCmdWithAllFlag(false), nil))
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.ElementsMatch(t, []string{"alpha", "beta"}, lines)
}

func TestListRun_AllFlagAlsoPrintsRequests(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "# @name listUsers\nGET /users\nHost: x\n", "svc", "http")

	out := captureStdout(t, func() {
		require.NoError(t, listRun(newListCmdWithAllFlag(true), nil))
	})
	assert.Contains(t, out, "svc")
	assert.Contains(t, out, "- listUsers")
}

func TestListRun_PositionalArgRestrictsAndAlwaysShowsRequests(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "other"), 0o755))
	makeFile(t, wd, "# @name pingTest\nGET /ping\nHost: x\n", "svc", "http")

	out := captureStdout(t, func() {
		require.NoError(t, listRun(newListCmdWithAllFlag(false), []string{"svc"}))
	})
	assert.Contains(t, out, "svc")
	assert.Contains(t, out, "- pingTest")
	assert.NotContains(t, out, "other", "positional arg must restrict output to the named namespace")
}

func TestListRun_PositionalUnknownNamespaceErrors(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "alpha"), 0o755))

	err := listRun(newListCmdWithAllFlag(false), []string{"missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestPrintNamespace_AllWithMissingHTTPFileSilentlySkipsRequests(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	out := captureStdout(t, func() { printNamespace(wd, "svc", true) })
	// Namespace name still prints; the missing http file is swallowed silently.
	assert.Contains(t, out, "svc")
	assert.NotContains(t, out, "- ")
}
