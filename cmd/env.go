/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/parikhrahil/httpgo/internal/config"
	"github.com/parikhrahil/httpgo/internal/utils"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env [namespace]",
	Short: "Print env variables visible to a namespace (or the shared globalenv)",
	Long: `Print KEY=VALUE pairs from the env files under ~/.httpgo/collections.

With no arguments, prints the shared "globalenv" file. With a namespace,
prints the resolved variable set that the namespace's requests see — that
is, "globalenv" merged with the namespace's "env" file, where namespace
values override global values for the same key.

  httpgo env             # globalenv only
  httpgo env appwf       # globalenv merged with appwf/env (appwf wins on conflicts)`,
	Args: cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureWorkingDirectory()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return envRun(args)
	},
}

func envRun(args []string) error {
	if len(args) == 0 {
		printEnv(utils.GetGlobalEnvVariables())
		return nil
	}

	ns := args[0]
	if !utils.IsValidNamespace(ns) {
		return fmt.Errorf("no collection found for namespace %q", ns)
	}
	printEnv(utils.GetEnvVariables(config.GetWorkingDirectory(), ns))
	return nil
}

func printEnv(vars map[string]string) {
	for k, v := range vars {
		fmt.Printf("%s=%s\n", k, v)
	}
}

func init() {
	rootCmd.AddCommand(envCmd)
}
