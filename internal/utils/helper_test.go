package utils_test

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parikhrahil/httpgo/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout swaps os.Stdout for a pipe while fn runs and returns whatever
// fn printed. utils.PrintRequest / PrintToConsole / WriteToFile write directly
// to fmt.Println (no injectable writer), so this is the only way to assert on
// their output.
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

func TestPrettyJson_FormatsValidJSON(t *testing.T) {
	in := []byte(`{"a":1,"b":[2,3]}`)
	got := utils.PrettyJson(in)
	assert.Contains(t, got, "  \"a\": 1")
	assert.Contains(t, got, "  \"b\": [")
}

func TestPrettyJson_ReturnsRawForInvalidJSON(t *testing.T) {
	// Suppress the "Error:" line that PrettyJson prints on bad input so it
	// doesn't muddy the test output.
	out := captureStdout(t, func() {
		got := utils.PrettyJson([]byte("not json"))
		assert.Equal(t, "not json", got, "invalid JSON must fall back to the original bytes")
	})
	assert.Contains(t, out, "Error:")
}

func TestWriteToFile_AppendsBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")

	out1 := captureStdout(t, func() { utils.WriteToFile(path, []byte("first")) })
	assert.Contains(t, out1, "Response saved to")

	out2 := captureStdout(t, func() { utils.WriteToFile(path, []byte("second")) })
	assert.Contains(t, out2, "Response saved to")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	// fmt.Fprintln adds a trailing newline per write.
	assert.Equal(t, "first\nsecond\n", string(data))
}

func TestWriteToFile_EmptyBodyDoesNotCreateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")

	out := captureStdout(t, func() { utils.WriteToFile(path, []byte{}) })
	assert.Contains(t, out, "No body to write")

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file should not be created for empty body")
}

func TestWriteToFile_OpenFailurePrintsError(t *testing.T) {
	// A path with a non-existent intermediate directory makes OpenFile fail.
	bad := filepath.Join(t.TempDir(), "missing-dir", "out.txt")

	out := captureStdout(t, func() { utils.WriteToFile(bad, []byte("payload")) })
	assert.Contains(t, out, "Could not open or create the file")
}

func TestPrintRequest_RestoresBodyAfterReading(t *testing.T) {
	bodyContent := `{"hello":"world"}`
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/users",
		strings.NewReader(bodyContent))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	out := captureStdout(t, func() { utils.PrintRequest(req) })
	assert.Contains(t, out, "Method: POST")
	assert.Contains(t, out, "URL: https://api.example.com/users")
	assert.Contains(t, out, "Content-Type: application/json")

	// Critical: the body must still be readable after PrintRequest returns,
	// or the caller's subsequent client.Do would send a zero-length body.
	got, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, bodyContent, string(got))
}

func TestPrintRequest_NilBodyDoesNotPanic(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/ping", nil)
	require.NoError(t, err)

	out := captureStdout(t, func() { utils.PrintRequest(req) })
	assert.Contains(t, out, "Method: GET")
	assert.NotContains(t, out, "Request Body:")
	assert.NotContains(t, out, "Request Form:")
}

func TestPrintRequest_FormEncodedBodyPrintsAsForm(t *testing.T) {
	form := url.Values{"name": {"alice"}, "role": {"admin"}}
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/login",
		strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	out := captureStdout(t, func() { utils.PrintRequest(req) })
	assert.Contains(t, out, "Request Form:")
	assert.Contains(t, out, "name: alice")
	assert.Contains(t, out, "role: admin")
	assert.NotContains(t, out, "Request Body:")
}

func TestPrintToConsole_DefaultPrintsStatusAndBody(t *testing.T) {
	res := &http.Response{Status: "200 OK", Header: http.Header{}}
	out := captureStdout(t, func() {
		utils.PrintToConsole(&utils.PrintProps{}, res, []byte("hello"))
	})
	assert.Contains(t, out, "Response Status: 200 OK")
	assert.Contains(t, out, "Response Body:")
	assert.Contains(t, out, "hello")
}

func TestPrintToConsole_RawSkipsAllChrome(t *testing.T) {
	res := &http.Response{Status: "200 OK", Header: http.Header{"X-Trace": []string{"abc"}}}
	out := captureStdout(t, func() {
		utils.PrintToConsole(&utils.PrintProps{Raw: true, Headers: true}, res, []byte("hello"))
	})
	assert.NotContains(t, out, "Response Status:", "raw mode must not print the status line")
	assert.NotContains(t, out, "Response Headers:", "raw mode must not print headers even when Headers=true")
	assert.Contains(t, out, "hello")
}

func TestPrintToConsole_HeadersIncludesResponseHeaders(t *testing.T) {
	res := &http.Response{
		Status: "200 OK",
		Header: http.Header{"X-Trace": []string{"abc123"}},
	}
	out := captureStdout(t, func() {
		utils.PrintToConsole(&utils.PrintProps{Headers: true}, res, []byte("ok"))
	})
	assert.Contains(t, out, "Response Headers:")
	assert.Contains(t, out, "X-Trace: abc123")
}

func TestPrintToConsole_PrettyFormatsJSON(t *testing.T) {
	res := &http.Response{Status: "200 OK", Header: http.Header{}}
	out := captureStdout(t, func() {
		utils.PrintToConsole(&utils.PrintProps{Pretty: true}, res, []byte(`{"a":1}`))
	})
	assert.Contains(t, out, "\"a\": 1")
}

func TestPrintToConsole_EmptyBodyMessage(t *testing.T) {
	res := &http.Response{Status: "204 No Content", Header: http.Header{}}
	out := captureStdout(t, func() {
		utils.PrintToConsole(&utils.PrintProps{}, res, nil)
	})
	assert.Contains(t, out, "[Empty Body]")
}

func TestIsValidNamespace(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(wd, "alpha"), 0o755))

	assert.True(t, utils.IsValidNamespace("alpha"))
	assert.False(t, utils.IsValidNamespace("beta"))
}

func TestIsValidNamespace_PanicsWhenWorkingDirMissing(t *testing.T) {
	// Pointing HOME at a tmp dir without pre-creating ~/.httpgo/collections
	// makes GetValidCollections fail, which IsValidNamespace turns into a panic.
	t.Setenv("HOME", t.TempDir())
	assert.Panics(t, func() { _ = utils.IsValidNamespace("anything") })
}
