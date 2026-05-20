/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"fmt"
	nethttp "net/http"
	"slices"

	"github.com/parikhrahil/httpgo/internal/config"
	"github.com/parikhrahil/httpgo/internal/http"
	"github.com/parikhrahil/httpgo/internal/utils"

	"github.com/spf13/cobra"
)

var collectionCmd = &cobra.Command{
	Use:     "collection <namespace> <request>",
	Aliases: []string{"cl", "run"},
	Short:   "Execute a named request from a namespace's http file",
	Long: `Execute a single named request from ~/.httpgo/collections/<namespace>/http.

Requests are separated by "###" lines and named via a "# @name foo" or
"// @name foo" comment inside the block. Every {KEY} placeholder is
replaced with the matching value from the namespace's env file (and the
shared globalenv) before the request is sent.

  httpgo collection appwf get-status
  httpgo run appwf get-status -v userId=42 -g host=https://api.test
  httpgo run appwf get-status -u userId -U host

Use --vars / --global-vars to upsert overrides into the namespace's env
file or the shared globalenv, respectively, before the request runs.
Use --unset / --global-unset to delete keys from those same files before
the request runs. Clears are applied before overrides, so the same key
can be cleared and re-set in one invocation.`,
	Args: cobra.MatchAll(cobra.ExactArgs(2), func(cmd *cobra.Command, args []string) error {
		return validateServiceArg(config.GetWorkingDirectory(), args[0])
	}),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return executeHTTP(cmd, args)
	},
}

func validateServiceArg(dir, namespace string) error {
	coll, err := utils.GetValidCollections(dir)
	if err != nil {
		return err
	}
	if len(coll) == 0 {
		return fmt.Errorf("no collection found in %s", dir)
	}
	if slices.Contains(coll, namespace) {
		return nil
	}
	return fmt.Errorf("no collection found for %s. Available collections: %v", namespace, coll)
}

func executeHTTP(cmd *cobra.Command, args []string) error {
	dir := config.GetWorkingDirectory()
	namespace := args[0]
	request := args[1]

	ctx, err := utils.NewCollectionContext(dir, namespace)
	if err != nil {
		return err
	}

	applyVarFlags(cmd, ctx)
	if err := ctx.Persist(); err != nil {
		return err
	}

	req, err := ctx.ParseNamedRequest(request)
	if err != nil {
		return err
	}

	raw, _ := cmd.Flags().GetBool("raw")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if !raw || dryRun {
		utils.PrintRequest(req)
	}

	if dryRun {
		return nil
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	res, body, err := http.ExecuteHTTPRequest(req, timeout)
	if err != nil {
		return err
	}

	writeToFile(cmd, body)
	printToConsole(cmd, res, body)
	return nil
}

// applyVarFlags applies --unset / --global-unset (clears) before
// --vars / --global-vars (overrides), so the same key can be cleared and
// re-set in a single invocation.
func applyVarFlags(cmd *cobra.Command, ctx *utils.CollectionContext) {
	if unset, _ := cmd.Flags().GetStringArray("unset"); len(unset) > 0 {
		ctx.ClearLocal(unset)
	}
	if globalUnset, _ := cmd.Flags().GetStringArray("global-unset"); len(globalUnset) > 0 {
		ctx.ClearGlobal(globalUnset)
	}
	if vars, _ := cmd.Flags().GetStringArray("vars"); len(vars) > 0 {
		ctx.OverrideLocal(vars)
	}
	if globalVars, _ := cmd.Flags().GetStringArray("global-vars"); len(globalVars) > 0 {
		ctx.OverrideGlobal(globalVars)
	}
}

func writeToFile(cmd *cobra.Command, body []byte) {
	outfile, _ := cmd.Flags().GetString("output")
	if outfile == "" {
		return
	}
	utils.WriteToFile(outfile, body)
}

func printToConsole(cmd *cobra.Command, res *nethttp.Response, body []byte) {
	raw, _ := cmd.Flags().GetBool("raw")
	headers, _ := cmd.Flags().GetBool("include-headers")
	prettify, _ := cmd.Flags().GetBool("prettify")

	utils.PrintToConsole(&utils.PrintProps{
		Raw:     raw,
		Pretty:  prettify,
		Headers: headers,
	}, res, body)
}

func init() {
	rootCmd.AddCommand(collectionCmd)

	f := collectionCmd.Flags()
	f.StringP("output", "o", "", "append the response body to this file (path)")
	f.Bool("dry-run", false, "resolve variables and print the request without sending it")
	f.BoolP("prettify", "p", true, "pretty-print JSON response bodies")
	f.BoolP("raw", "r", false, "print only the response body (suppresses request dump and status line)")
	f.BoolP("include-headers", "H", false, "include response headers in the printed output")
	f.DurationP("timeout", "t", 0, "per-request timeout, e.g. 5s or 500ms")
	f.StringArrayP("vars", "v", nil, "upsert KEY=VALUE into the namespace's env file before running (repeatable)")
	f.StringArrayP("global-vars", "g", nil, "upsert KEY=VALUE into the shared globalenv before running (repeatable)")
	f.StringArrayP("unset", "u", nil, "delete KEY from the namespace's env file before running (repeatable)")
	f.StringArrayP("global-unset", "U", nil, "delete KEY from the shared globalenv before running (repeatable)")
	collectionCmd.MarkFlagFilename("output")
}
