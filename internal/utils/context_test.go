package utils_test

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/parikhrahil/httpgo/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollectionContext_MissingEnvFilesAreTreatedAsEmpty(t *testing.T) {
	wd := setupHTTPGoHome(t)
	// Create a namespace directory but no env file inside it.
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)
	assert.Empty(t, ctx.Env(), "missing globalenv and namespace env should yield empty merged env")
}

func TestNewCollectionContext_MalformedGlobalEnvErrors(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("malformed-line-without-equals\n"), 0o644))

	_, err := utils.NewCollectionContext(wd, "svc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load global env")
}

func TestNewCollectionContext_MalformedNamespaceEnvErrors(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "malformed-line-without-equals\n", "svc", "env")

	_, err := utils.NewCollectionContext(wd, "svc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load svc env")
}

func TestEnv_NamespaceOverridesGlobal(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("HOST=global\nSHARED=g\n"), 0o644))
	makeFile(t, wd, "HOST=local\nNS_ONLY=ns\n", "svc", "env")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"HOST":    "local",
		"SHARED":  "g",
		"NS_ONLY": "ns",
	}, ctx.Env())
}

func TestOverrideLocal_AndPersist_WritesNamespaceEnvOnce(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "EXISTING=old\n", "svc", "env")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.OverrideLocal([]string{"EXISTING=new", "FRESH=value"})
	require.NoError(t, ctx.Persist())

	got := readEnvFile(t, filepath.Join(wd, "svc", "env"))
	assert.Equal(t, map[string]string{
		"EXISTING": "new",
		"FRESH":    "value",
	}, got)
}

func TestOverrideGlobal_AndPersist_WritesGlobalEnv(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"), []byte("A=1\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.OverrideGlobal([]string{"A=2", "B=3"})
	require.NoError(t, ctx.Persist())

	got := readEnvFile(t, filepath.Join(wd, "globalenv"))
	assert.Equal(t, map[string]string{"A": "2", "B": "3"}, got)
}

func TestClearLocal_RemovesKeys(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "KEEP=yes\nDROP=bye\n", "svc", "env")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.ClearLocal([]string{"DROP", "DOES_NOT_EXIST"})
	require.NoError(t, ctx.Persist())

	got := readEnvFile(t, filepath.Join(wd, "svc", "env"))
	assert.Equal(t, map[string]string{"KEEP": "yes"}, got)
}

func TestClearGlobal_RemovesKeys(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("KEEP=yes\nDROP=bye\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.ClearGlobal([]string{"DROP"})
	require.NoError(t, ctx.Persist())

	got := readEnvFile(t, filepath.Join(wd, "globalenv"))
	assert.Equal(t, map[string]string{"KEEP": "yes"}, got)
}

func TestClearThenOverride_SameKey_ReplacesValue(t *testing.T) {
	// Mirrors the documented invocation order in cmd/collection.go: clears
	// first, overrides second. The same key can be cleared and re-set in one
	// run.
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "TOKEN=old\n", "svc", "env")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.ClearLocal([]string{"TOKEN"})
	ctx.OverrideLocal([]string{"TOKEN=fresh"})
	require.NoError(t, ctx.Persist())

	got := readEnvFile(t, filepath.Join(wd, "svc", "env"))
	assert.Equal(t, map[string]string{"TOKEN": "fresh"}, got)
}

func TestPersist_NoOpWhenNothingDirty(t *testing.T) {
	wd := setupHTTPGoHome(t)
	original := []byte("X=preserved\n")
	envPath := filepath.Join(wd, "svc", "env")
	require.NoError(t, os.MkdirAll(filepath.Dir(envPath), 0o755))
	require.NoError(t, os.WriteFile(envPath, original, 0o644))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	// Empty override pairs and clearing absent keys must NOT dirty the maps.
	ctx.OverrideLocal(nil)
	ctx.OverrideGlobal([]string{})
	ctx.ClearLocal([]string{"NOT_PRESENT"})
	ctx.ClearGlobal([]string{"NOT_PRESENT"})
	require.NoError(t, ctx.Persist())

	// File content must be byte-identical: Persist must not rewrite a clean file.
	got, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, original, got, "Persist rewrote a clean env file")
}

func TestPersist_IsIdempotentOnDirtyThenClean(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.OverrideLocal([]string{"K=v"})
	require.NoError(t, ctx.Persist())

	// Modify the file externally; a second Persist must not stomp it (dirty flag cleared).
	envPath := filepath.Join(wd, "svc", "env")
	external := []byte("EXTERNAL=write\n")
	require.NoError(t, os.WriteFile(envPath, external, 0o644))

	require.NoError(t, ctx.Persist())

	got, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, external, got, "Persist must not rewrite once the dirty flag is cleared")
}

func TestParseNamedRequest_InterpolatesVariablesAndDefaultsHTTPVersion(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"), []byte("HOST=api.example.com\n"), 0o644))
	httpFile := "# @name listUsers\n" +
		"GET /users\n" +
		"Host: {HOST}\n" +
		"\n" +
		"###\n" +
		"# @name ping\n" +
		"GET /ping\n" +
		"Host: {HOST}\n"
	makeFile(t, wd, httpFile, "svc", "http")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	// First block (separated by ###).
	req, err := ctx.ParseNamedRequest("listUsers")
	require.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/users", req.URL.Path)
	assert.Equal(t, "api.example.com", req.URL.Host, "URL.Host should be rebuilt from Host header")
	assert.Equal(t, "http", req.URL.Scheme)
	assert.Equal(t, "HTTP/1.1", req.Proto, "missing HTTP version should default to HTTP/1.1")

	// Last block in the file (no trailing ###) — exercises the "catch the last block" branch.
	req2, err := ctx.ParseNamedRequest("ping")
	require.NoError(t, err)
	assert.Equal(t, "/ping", req2.URL.Path)
	assert.Equal(t, "api.example.com", req2.URL.Host)
}

func TestParseNamedRequest_InjectsContentLengthForPOSTBody(t *testing.T) {
	wd := setupHTTPGoHome(t)
	body := `{"name":"alice"}`
	httpFile := "# @name createUser\n" +
		"POST /users\n" +
		"Host: api.example.com\n" +
		"Content-Type: application/json\n" +
		"\n" +
		body + "\n"
	makeFile(t, wd, httpFile, "svc", "http")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	req, err := ctx.ParseNamedRequest("createUser")
	require.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, int64(len(body)), req.ContentLength,
		"Content-Length must be injected from the parsed body when the header is absent")

	got, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))
}

func TestParseNamedRequest_RespectsExistingContentLengthHeader(t *testing.T) {
	wd := setupHTTPGoHome(t)
	body := `{"name":"bob"}`
	httpFile := "# @name createUser\n" +
		"POST /users\n" +
		"Host: api.example.com\n" +
		"Content-Type: application/json\n" +
		"Content-Length: " + strconv.Itoa(len(body)) + "\n" +
		"\n" +
		body + "\n"
	makeFile(t, wd, httpFile, "svc", "http")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	req, err := ctx.ParseNamedRequest("createUser")
	require.NoError(t, err)
	assert.Equal(t, int64(len(body)), req.ContentLength)
	// Header should appear exactly once.
	all := req.Header.Values("Content-Length")
	assert.LessOrEqual(t, len(all), 1, "Content-Length should not be duplicated when already present")
}

func TestParseNamedRequest_UnknownNameReturnsError(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "# @name only\nGET /x\nHost: a.example.com\n", "svc", "http")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	_, err = ctx.ParseNamedRequest("doesNotExist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesNotExist")
}

func TestParseNamedRequest_OverridesAreVisibleWithoutPersist(t *testing.T) {
	// Confirms env state is in-memory: overrides must affect interpolation even
	// when Persist() is never called.
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "HOST=disk.example.com\n", "svc", "env")
	makeFile(t, wd, "# @name get\nGET /\nHost: {HOST}\n", "svc", "http")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	ctx.OverrideLocal([]string{"HOST=memory.example.com"})

	req, err := ctx.ParseNamedRequest("get")
	require.NoError(t, err)
	assert.Equal(t, "memory.example.com", req.URL.Host)
}

// readEnvFile parses a KEY=value file written by writeEnvFile.
// The on-disk format is deterministic per line but map iteration order is not,
// so callers must compare via maps, not byte-equality.
func readEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		require.True(t, ok, "malformed env line: %q", line)
		out[k] = v
	}
	return out
}

