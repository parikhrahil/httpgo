package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWdCmd_PrintsCollectionsDirectory(t *testing.T) {
	wd := setupHTTPGoHome(t)

	out := captureStdout(t, func() { wdCmd.Run(wdCmd, nil) })
	assert.Equal(t, wd, strings.TrimSpace(out))
}
