package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taylormonacelli/fragiledonkey/cleanup"
)

var (
	olderThan string
	newerThan string
	assumeYes bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup AMIs and snapshots based on relative date",
	Run: func(cmd *cobra.Command, args []string) {
		if olderThan == "" && newerThan == "" {
			fmt.Println("Error: either --older-than or --newer-than must be provided")
			err := cmd.Help()
			if err != nil {
				fmt.Println("Error displaying help:", err)
			}
			return
		}
		cleanup.RunCleanup(olderThan, newerThan, assumeYes)
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVar(&olderThan, "older-than", "", "Relative date for cleanup (e.g., 7d, 1M)")
	cleanupCmd.Flags().StringVar(&newerThan, "newer-than", "", "Relative date for cleanup (e.g., 7d, 1M)")
	cleanupCmd.MarkFlagsMutuallyExclusive("older-than", "newer-than")
	cleanupCmd.Flags().BoolVar(&assumeYes, "assume-yes", false, "Assume yes to prompts and run non-interactively")
}
