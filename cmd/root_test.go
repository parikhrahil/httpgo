package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveVersion_ReturnsFallbackForDevelBuild(t *testing.T) {
	// `go test` produces a debug.BuildInfo whose Main.Version is "(devel)";
	// resolveVersion must replace that placeholder with a friendlier string.
	got := resolveVersion()
	assert.Equal(t, "unknown (built from source)", got)
}
