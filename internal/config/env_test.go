package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parikhrahil/httpgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkingDirectory(t *testing.T) {
	wd := config.GetWorkingDirectory()
	assert.True(t, strings.HasSuffix(wd, ".httpgo/collections"),
		"expected suffix .httpgo/collections, got %s", wd)
	assert.True(t, filepath.IsAbs(wd), "expected absolute path, got %s", wd)
}

func TestGetGlobalEnvFile(t *testing.T) {
	wd := config.GetWorkingDirectory()
	gf := config.GetGlobalEnvFile()
	assert.Equal(t, filepath.Join(wd, "globalenv"), gf)
}

func TestLoad_BasicFile(t *testing.T) {
	got, err := config.Load("./testdata/basic.env")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"FOO": "bar",
		"BAZ": "qux",
	}, got)
}

func TestLoad_StripsCommentsAndKeepsHashInsideValues(t *testing.T) {
	got, err := config.Load("./testdata/comments.env")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"HOST":  "localhost",
		"PORT":  "8080",
		"TOKEN": "abc",
	}, got, "inline comments must be stripped even from unquoted values")
}

func TestLoad_QuotedValues(t *testing.T) {
	got, err := config.Load("./testdata/quoted.env")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"PLAIN":       "hello",
		"WITH_SPACES": "hello world",
		"WITH_INLINE": "hello world",
		"WITH_HASH":   "value#with#hash",
	}, got, "quoted segment is preserved verbatim and trailing comment is dropped")
}

func TestLoad_WhitespaceAndTabs(t *testing.T) {
	got, err := config.Load("./testdata/whitespace.env")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"PADDED": "spaced value",
		"TAB":    "tabbed",
	}, got)
}

func TestLoad_EmptyFile(t *testing.T) {
	got, err := config.Load("./testdata/empty.env")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLoad_LaterFileOverridesEarlier(t *testing.T) {
	got, err := config.Load("./testdata/basic.env", "./testdata/override.env")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"FOO": "overridden",
		"BAZ": "qux",
		"NEW": "fresh",
	}, got)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("./testdata/does-not-exist.env")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "expected fs.ErrNotExist, got %v", err)
}

func TestLoad_MalformedLineReturnsError(t *testing.T) {
	_, err := config.Load("./testdata/bad.env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}
