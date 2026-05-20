/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/parikhrahil/httpgo/internal/config"

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
}

// Execute is the entrypoint invoked by main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	wd := config.GetWorkingDirectory()

	// 0755: rwx for owner, rx for group/others.
	if err := os.MkdirAll(wd, 0755); err != nil {
		fmt.Printf("Failed to create directory %s. Please create manually\n", wd)
	}

	// Touch globalenv if it doesn't already exist; O_EXCL preserves any
	// existing contents. Ignore the "already exists" error.
	f, err := os.OpenFile(config.GetGlobalEnvFile(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err == nil {
		f.Close()
	}
}
