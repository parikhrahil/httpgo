/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "httpgo",
	Short: "Scriptable HTTP client driven by .http files on disk",
	Long: `httpgo runs named HTTP requests stored under ~/.httpgo/collections.

A collection is a directory of service ("namespace") subdirectories. Each
namespace holds two files (literally named "http" and "env"):

  ~/.httpgo/collections/
    globalenv        # KEY=value variables shared across all namespaces
    appwf/
      http           # one or more "###"-separated named request blocks
      env            # KEY=value variables scoped to this namespace

Variables are referenced inside request blocks as {KEY}. Values from a
namespace's env file override values from globalenv for the same key.

Commands:
  collection (cl, run)  Execute a named request in a namespace
  list       (ls)       List namespaces (and optionally their requests)
  env                   Print the variables visible to a namespace (or globalenv)
  wd                    Print the collections working directory`,
	Version: resolveVersion(),
}

// Execute is the entrypoint invoked by main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resolveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "unknown (built from source)"
	}
	return info.Main.Version
}
