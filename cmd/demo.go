// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Demo progress bar functionality",
	Long:  `Show how the progress bar works when processing many files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Simulating processing 1000 files...")

		bar := progressbar.Default(1000, "Processing files")
		for range 1000 {
			time.Sleep(2 * time.Millisecond) // Simulate work
			_ = bar.Add(1)
		}

		fmt.Println("\n✓ Demo completed!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(demoCmd)
}
