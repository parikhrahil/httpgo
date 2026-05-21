/*
Copyright © 2026 Rahil Parikh <rahilparikh11@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/parikhrahil/httpgo/internal/config"

	"github.com/spf13/cobra"
)

// wdCmd represents the wd command
var wdCmd = &cobra.Command{
	Use:   "wd",
	Short: "Print the collections working directory",
	Long: `Print the absolute path of the collections working directory
(~/.httpgo/collections) that every other command reads from.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureWorkingDirectory()
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.GetWorkingDirectory())
	},
}

func init() {
	rootCmd.AddCommand(wdCmd)
}
