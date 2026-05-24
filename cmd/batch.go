/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/parikhrahil/httpgo/internal/batch"
	"github.com/parikhrahil/httpgo/internal/config"
	"github.com/spf13/cobra"
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.MatchAll(cobra.ExactArgs(2), func(cmd *cobra.Command, args []string) error {
		return validateServiceArg(config.GetWorkingDirectory(), args[0])
	}),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		infile, _ := cmd.Flags().GetString("input")
		if ft := filepath.Ext(infile); ft != ".json" && ft != ".csv" {
			return fmt.Errorf("Input file type should be one of csv or json")
		}
		outfile, _ := cmd.Flags().GetString("output")
		if ft := filepath.Ext(outfile); outfile != "" && ft != ".csv" {
			return fmt.Errorf("Output file type should be one of csv or json")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return batchRun(cmd, args)
	},
}

func batchRun(cmd *cobra.Command, args []string) error {
	dir := config.GetWorkingDirectory()
	namespace := args[0]
	request := args[1]

	collCtx, err := CollectionContext(cmd, dir, namespace)
	if err != nil {
		return err
	}

	file, _ := cmd.Flags().GetString("input")
	concurrency, _ := cmd.Flags().GetUint("concurrency")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	output, _ := cmd.Flags().GetString("output")
	includeBody, _ := cmd.Flags().GetBool("include-body")

	ctx, err := batch.NewContext(&batch.BatchOpts{
		Request:     request,
		Filepath:    file,
		Concurrency: concurrency,
		Timeout:     timeout,
		DryRun:      dryRun,
		OutFile:     output,
		IncludeBody: includeBody,
	})
	return ctx.Execute(collCtx)
}

func init() {
	collectionCmd.AddCommand(batchCmd)

	f := batchCmd.Flags()

	f.StringP("input", "i", "", "input file for batch processing")
	f.UintP("concurrency", "n", 0, "concurrency for batch processing")
	f.StringP("output", "o", "", "output file")
	f.Bool("include-body", false, "include response body in output file")

	batchCmd.MarkFlagRequired("input")
	batchCmd.MarkFlagFilename("input", ".csv", ".json")
	batchCmd.MarkFlagFilename("output", ".csv", ".json")
}
