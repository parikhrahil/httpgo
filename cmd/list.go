/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"fmt"
	"slices"

	"github.com/parikhrahil/httpgo/internal/config"
	"github.com/parikhrahil/httpgo/internal/utils"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list [namespace]",
	Aliases: []string{"ls"},
	Short:   "List namespaces in the collections working directory",
	Long: `List the namespaces available under ~/.httpgo/collections.

By default only namespace names are printed. Use --all to also list every
named request inside each namespace, or pass a namespace as a positional
arg to restrict output to a single namespace (its requests are always shown).

  httpgo list
  httpgo ls --all
  httpgo ls appwf`,
	Args: cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureWorkingDirectory()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return listRun(cmd, args)
	},
}

func listRun(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")
	wd := config.GetWorkingDirectory()

	namespaces, err := utils.GetValidCollections(wd)
	if err != nil {
		return fmt.Errorf("could not fetch available namespaces from %s", wd)
	}

	if len(args) > 0 {
		namespace := args[0]
		if !slices.Contains(namespaces, namespace) {
			return fmt.Errorf("no collection found for %s", namespace)
		}
		printNamespace(wd, namespace, true)
		return nil
	}

	for _, n := range namespaces {
		printNamespace(wd, n, all)
	}
	return nil
}

func printNamespace(wd, namespace string, all bool) {
	fmt.Println(namespace)
	if !all {
		return
	}
	requests, err := utils.GetRequestsForNamespace(wd, namespace)
	if err != nil {
		return
	}
	for _, r := range requests {
		fmt.Printf("- %s\n", r)
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("all", "a", false, "also print every named request under each namespace")
}
