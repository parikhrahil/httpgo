package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/parikhrahil/httpgo/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeFile writes content to <dir>/<elems...> and creates parent dirs.
func makeFile(t *testing.T, dir string, content string, elems ...string) string {
	t.Helper()
	parts := append([]string{dir}, elems...)
	full := filepath.Join(parts...)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	return full
}

// setupHTTPGoHome points HOME at a fresh tmp dir and pre-creates
// <tmp>/.httpgo/collections, the layout the binary uses. Returns the
// collections dir so callers can build namespaces under it.
func setupHTTPGoHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := filepath.Join(home, ".httpgo", "collections")
	require.NoError(t, os.MkdirAll(wd, 0o755))
	return wd
}

func TestGetValidCollections_ReturnsOnlyDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "alpha"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "beta"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("x"), 0o644))

	got, err := utils.GetValidCollections(dir)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"alpha", "beta"}, got)
}

func TestGetValidCollections_EmptyDir(t *testing.T) {
	got, err := utils.GetValidCollections(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetValidCollections_MissingDir(t *testing.T) {
	_, err := utils.GetValidCollections(filepath.Join(t.TempDir(), "nope"))
	require.Error(t, err)
}

func TestGetRequestsForNamespace_ExtractsBothCommentStyles(t *testing.T) {
	dir := t.TempDir()
	content := `### separator only, must not register as a name
# @name listUsers
GET https://example.com/users

###
// @name createUser
POST https://example.com/users

###
# this is a plain comment with no @name annotation
GET https://example.com/ping
`
	makeFile(t, dir, content, "svc", "http")

	got, err := utils.GetRequestsForNamespace(dir, "svc")
	require.NoError(t, err)
	assert.Equal(t, []string{"listUsers", "createUser"}, got)
}

func TestGetRequestsForNamespace_MissingFileErrors(t *testing.T) {
	_, err := utils.GetRequestsForNamespace(t.TempDir(), "no-such-namespace")
	require.Error(t, err)
}

func TestGetRequestsForNamespace_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	makeFile(t, dir, "", "svc", "http")

	got, err := utils.GetRequestsForNamespace(dir, "svc")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetGlobalEnvVariables_ReadsGlobalEnvFile(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("HOST=global.example.com\nTOKEN=topsecret\n"), 0o644))

	got := utils.GetGlobalEnvVariables()
	assert.Equal(t, map[string]any{
		"HOST":  "global.example.com",
		"TOKEN": "topsecret",
	}, got)
}

func TestGetGlobalEnvVariables_MissingFileReturnsNil(t *testing.T) {
	// HOME points at a tmp dir, but globalenv is intentionally absent.
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := utils.GetGlobalEnvVariables()
	assert.Nil(t, got, "Load swallows the error and returns nil when the file is absent")
}

func TestGetEnvVariables_LocalOverridesGlobal(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("HOST=global.example.com\nSHARED=from-global\n"), 0o644))

	// Namespace env shadows HOST and adds NS_ONLY.
	makeFile(t, wd, "HOST=ns.example.com\nNS_ONLY=local\n", "svc", "env")

	got := utils.GetEnvVariables(wd, "svc")
	assert.Equal(t, map[string]any{
		"HOST":    "ns.example.com",
		"SHARED":  "from-global",
		"NS_ONLY": "local",
	}, got)
}

func TestGetEnvVariables_NoLocalEnvFallsBackToGlobal(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("HOST=global.example.com\n"), 0o644))

	got := utils.GetEnvVariables(wd, "ns-without-env-file")
	assert.Equal(t, map[string]any{"HOST": "global.example.com"}, got)
}
