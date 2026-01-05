// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/YakDriver/copyplop/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of copyplop",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("v%s\n", version.Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
