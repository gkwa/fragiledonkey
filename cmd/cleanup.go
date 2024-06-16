package cmd

import (
	"fmt"

	"github.com/gkwa/fragiledonkey/cleanup"
	"github.com/spf13/cobra"
)

var (
	olderThan      string
	newerThan      string
	assumeYes      bool
	leaveCountFlag int
	pattern        string
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup AMIs and snapshots based on relative date",
	Run: func(cmd *cobra.Command, args []string) {
		if olderThan == "" && newerThan == "" && leaveCountFlag == 0 {
			fmt.Println("Error: either --older-than, --newer-than, or --leave-count-remaining must be provided")
			err := cmd.Help()
			if err != nil {
				fmt.Println("Error displaying help:", err)
			}
			return
		}
		cleanup.RunCleanup(olderThan, newerThan, assumeYes, leaveCountFlag, pattern)
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVar(&olderThan, "older-than", "", "Relative date for cleanup (e.g., 7d, 1M)")
	cleanupCmd.Flags().StringVar(&newerThan, "newer-than", "", "Relative date for cleanup (e.g., 7d, 1M)")
	cleanupCmd.Flags().BoolVarP(&assumeYes, "assume-yes", "y", false, "Assume yes to prompts and run non-interactively")
	cleanupCmd.Flags().IntVar(&leaveCountFlag, "leave-count-remaining", 0, "Number of newest AMIs to keep")
	cleanupCmd.Flags().StringVar(&pattern, "pattern", "northflier-????-??-??-*", "Pattern for matching AMI names")
}
