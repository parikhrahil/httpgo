package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/parikhrahil/httpgo/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeHTTPBlock writes a one-block http file at <wd>/<ns>/http that targets
// targetURL. The block is named name and supports the "{KEY}" placeholders
// surrounding the URL so callers can verify variable interpolation.
func writeHTTPBlock(t *testing.T, wd, ns, name, method, targetURL, body string) {
	t.Helper()
	// Derive Host and path from the test server URL.
	u := targetURL
	hostStart := strings.Index(u, "://") + 3
	rest := u[hostStart:]
	slash := strings.Index(rest, "/")
	var host, path string
	if slash == -1 {
		host = rest
		path = "/"
	} else {
		host = rest[:slash]
		path = rest[slash:]
	}

	block := fmt.Sprintf("# @name %s\n%s %s\nHost: %s\n", name, method, path, host)
	if body != "" {
		block += "Content-Type: application/json\n\n" + body + "\n"
	}
	makeFile(t, wd, block, ns, "http")
}

func TestValidateServiceArg_MissingWorkingDirErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := validateServiceArg(filepath.Join(t.TempDir(), "no-such-dir"), "svc")
	require.Error(t, err)
}

func TestValidateServiceArg_EmptyWorkingDirErrors(t *testing.T) {
	wd := setupHTTPGoHome(t)
	err := validateServiceArg(wd, "svc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no collection found in")
}

func TestValidateServiceArg_UnknownNamespaceListsAvailable(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "alpha"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "beta"), 0o755))

	err := validateServiceArg(wd, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
	assert.Contains(t, err.Error(), "alpha")
	assert.Contains(t, err.Error(), "beta")
}

func TestValidateServiceArg_ValidNamespace(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))
	require.NoError(t, validateServiceArg(wd, "svc"))
}

func TestApplyVarFlags_ClearsRunBeforeOverrides(t *testing.T) {
	// cmd/collection.go documents that clears run before overrides so the
	// same key can be cleared and re-set in one invocation. This test pins
	// that ordering: when both --unset TOKEN and --vars TOKEN=fresh are
	// passed, the final value must be "fresh".
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "TOKEN=stale\n", "svc", "env")

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	c := newCollectionTestCmd()
	require.NoError(t, c.PersistentFlags().Set("unset", "TOKEN"))
	require.NoError(t, c.PersistentFlags().Set("vars", "TOKEN=fresh"))

	applyVarFlags(c, ctx)
	assert.Equal(t, map[string]any{"TOKEN": "fresh"}, ctx.Env())
}

func TestApplyVarFlags_GlobalCleanupAndOverride(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("DROP=bye\nKEEP=ok\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	ctx, err := utils.NewCollectionContext(wd, "svc")
	require.NoError(t, err)

	c := newCollectionTestCmd()
	require.NoError(t, c.PersistentFlags().Set("global-unset", "DROP"))
	require.NoError(t, c.PersistentFlags().Set("global-vars", "NEW=fresh"))

	applyVarFlags(c, ctx)
	assert.Equal(t, map[string]any{"KEEP": "ok", "NEW": "fresh"}, ctx.Env())
}

func TestExecuteHTTP_HappyPathSendsRequestAndPrints(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	wd := setupHTTPGoHome(t)
	writeHTTPBlock(t, wd, "svc", "fetch", "GET", srv.URL, "")

	c := newCollectionTestCmd()
	out := captureStdout(t, func() {
		require.NoError(t, collectionRun(c, []string{"svc", "fetch"}))
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "request should hit the test server exactly once")
	assert.Contains(t, out, "Response Status: 200 OK")
	assert.Contains(t, out, `"hello"`, "response body should be printed")
	assert.Contains(t, out, "Method: GET", "request dump should appear when --raw is unset")
}

func TestExecuteHTTP_DryRunSkipsNetworkAndPrintsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("dry-run must not hit the network")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	wd := setupHTTPGoHome(t)
	writeHTTPBlock(t, wd, "svc", "fetch", "GET", srv.URL, "")

	c := newCollectionTestCmd()
	require.NoError(t, c.PersistentFlags().Set("dry-run", "true"))

	out := captureStdout(t, func() {
		require.NoError(t, collectionRun(c, []string{"svc", "fetch"}))
	})
	assert.Contains(t, out, "Method: GET")
	assert.NotContains(t, out, "Response Status:", "dry-run must not produce a response section")
}

func TestExecuteHTTP_OutputFlagWritesBodyToFileAndSuppressesConsole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	wd := setupHTTPGoHome(t)
	writeHTTPBlock(t, wd, "svc", "fetch", "GET", srv.URL, "")

	outfile := filepath.Join(t.TempDir(), "out.bin")
	c := newCollectionTestCmd()
	require.NoError(t, c.Flags().Set("output", outfile))

	console := captureStdout(t, func() {
		require.NoError(t, collectionRun(c, []string{"svc", "fetch"}))
	})

	// --output skips the console "Response Status:" path entirely; only the
	// "Response saved to" notice is printed.
	assert.NotContains(t, console, "Response Status:")
	assert.Contains(t, console, "Response saved to")

	got, err := os.ReadFile(outfile)
	require.NoError(t, err)
	assert.Equal(t, "payload\n", string(got), "WriteToFile appends a trailing newline via Fprintln")
}

func TestExecuteHTTP_TeeFlagWritesBodyAndPrintsToConsole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("teed"))
	}))
	defer srv.Close()

	wd := setupHTTPGoHome(t)
	writeHTTPBlock(t, wd, "svc", "fetch", "GET", srv.URL, "")

	teefile := filepath.Join(t.TempDir(), "tee.bin")
	c := newCollectionTestCmd()
	require.NoError(t, c.Flags().Set("tee", teefile))

	out := captureStdout(t, func() {
		require.NoError(t, collectionRun(c, []string{"svc", "fetch"}))
	})

	// --tee prints AND writes.
	assert.Contains(t, out, "Response Status: 200 OK")
	assert.Contains(t, out, "teed")
	assert.Contains(t, out, "Response saved to")

	got, err := os.ReadFile(teefile)
	require.NoError(t, err)
	assert.Equal(t, "teed\n", string(got))
}

func TestExecuteHTTP_VarsFlagPersistsAndInterpolates(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wd := setupHTTPGoHome(t)
	hostStart := strings.Index(srv.URL, "://") + 3
	host := srv.URL[hostStart:]

	// The http block uses {PATH} which is supplied via --vars on this run.
	httpFile := "# @name fetch\nGET {PATH}\nHost: " + host + "\n"
	makeFile(t, wd, httpFile, "svc", "http")

	c := newCollectionTestCmd()
	require.NoError(t, c.PersistentFlags().Set("vars", "PATH=/things/42"))

	captureStdout(t, func() {
		require.NoError(t, collectionRun(c, []string{"svc", "fetch"}))
	})

	assert.Equal(t, "/things/42", gotPath)

	// --vars must also have been persisted to the namespace env file via ctx.Persist().
	envBytes, err := os.ReadFile(filepath.Join(wd, "svc", "env"))
	require.NoError(t, err)
	assert.Contains(t, string(envBytes), "PATH=/things/42")
}

func TestExecuteHTTP_UnknownRequestReturnsError(t *testing.T) {
	wd := setupHTTPGoHome(t)
	makeFile(t, wd, "# @name only\nGET /\nHost: example.com\n", "svc", "http")

	c := newCollectionTestCmd()
	// Suppress the partial request dump that prints before ParseNamedRequest fails.
	captureStdout(t, func() {
		err := collectionRun(c, []string{"svc", "missing"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})
}

func TestExecuteHTTP_NewContextFailureBubblesUp(t *testing.T) {
	wd := setupHTTPGoHome(t)
	// Malformed globalenv → NewCollectionContext fails before the request
	// file is even read.
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("malformed-line-without-equals\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "svc"), 0o755))

	c := newCollectionTestCmd()
	err := collectionRun(c, []string{"svc", "anything"})
	require.Error(t, err)
}

func TestExecuteHTTP_NetworkErrorReturnsError(t *testing.T) {
	wd := setupHTTPGoHome(t)
	// Point at 127.0.0.1:1 (nothing listens there) with a short timeout.
	writeHTTPBlock(t, wd, "svc", "fetch", "GET", "http://127.0.0.1:1/", "")

	c := newCollectionTestCmd()
	require.NoError(t, c.PersistentFlags().Set("timeout", "200ms"))

	captureStdout(t, func() {
		err := collectionRun(c, []string{"svc", "fetch"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network request failed")
	})
}

// Sanity-check that the test server URLs we feed into writeHTTPBlock parse
// into Host + path correctly. Guards against accidental drift in the helper.
func TestWriteHTTPBlockHelper(t *testing.T) {
	wd := t.TempDir()
	writeHTTPBlock(t, wd, "svc", "fetch", "POST", "http://127.0.0.1:8080/api/v1", `{"x":1}`)
	got, err := io.ReadAll(mustOpen(t, filepath.Join(wd, "svc", "http")))
	require.NoError(t, err)
	s := string(got)
	assert.Contains(t, s, "POST /api/v1")
	assert.Contains(t, s, "Host: 127.0.0.1:8080")
	assert.Contains(t, s, `{"x":1}`)
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}
