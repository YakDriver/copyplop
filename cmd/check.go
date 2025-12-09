package cmd

import (
	"fmt"
	"os"

	"github.com/YakDriver/copyplop/internal/copyright"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for missing or incorrect copyright headers",
	Long:  `Scan files and report any missing or incorrect copyright headers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := viper.GetString("path")

		checker := copyright.NewChecker(cfg)
		issues, err := checker.Check(path)
		if err != nil {
			return fmt.Errorf("check failed: %w", err)
		}

		if len(issues) > 0 {
			for _, issue := range issues {
				fmt.Printf("%s: %s\n", issue.File, issue.Problem)
			}
			fmt.Printf("\nFound %d files with copyright issues\n", len(issues))
			os.Exit(1)
		}

		fmt.Println("✓ All files have correct copyright headers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
