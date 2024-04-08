package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taylormonacelli/fragiledonkey/cleanup"
)

var (
	olderThan string
	assumeYes bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup AMIs and snapshots older than specified relative date",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cleaning up AMIs and snapshots...")
		cleanup.RunCleanup(olderThan, assumeYes)
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVar(&olderThan, "older-than", "", "Relative date for cleanup (e.g., 7d, 1M)")
	err := cleanupCmd.MarkFlagRequired("older-than")
	if err != nil {
		panic(err)
	}
	cleanupCmd.Flags().BoolVar(&assumeYes, "assume-yes", false, "Assume yes to prompts and run non-interactively")
}
