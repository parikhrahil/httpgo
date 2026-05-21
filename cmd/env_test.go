package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvRun_NoArgsPrintsGlobalEnv(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("FOO=bar\nBAZ=qux\n"), 0o644))

	out := captureStdout(t, func() {
		require.NoError(t, envRun(nil))
	})
	assert.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, parseEnvOutput(out))
}

func TestEnvRun_NoArgsNoGlobalEnvProducesEmptyOutput(t *testing.T) {
	setupHTTPGoHome(t)
	out := captureStdout(t, func() {
		require.NoError(t, envRun(nil))
	})
	assert.Empty(t, parseEnvOutput(out))
}

func TestEnvRun_NamespacePrintsMergedEnv(t *testing.T) {
	wd := setupHTTPGoHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(wd, "globalenv"),
		[]byte("HOST=global\nSHARED=g\n"), 0o644))
	makeFile(t, wd, "HOST=local\nNS_ONLY=ns\n", "svc", "env")

	out := captureStdout(t, func() {
		require.NoError(t, envRun([]string{"svc"}))
	})
	assert.Equal(t, map[string]string{
		"HOST":    "local",
		"SHARED":  "g",
		"NS_ONLY": "ns",
	}, parseEnvOutput(out))
}

func TestEnvRun_UnknownNamespaceErrors(t *testing.T) {
	setupHTTPGoHome(t)
	err := envRun([]string{"does-not-exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}
