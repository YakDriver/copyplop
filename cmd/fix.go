package cmd

import (
	"fmt"

	"github.com/YakDriver/copyplop/internal/copyright"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix missing or incorrect copyright headers",
	Long:  `Add or update copyright headers in source code files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := viper.GetString("path")

		fixer := copyright.NewFixer(cfg)
		results, err := fixer.Fix(path)
		if err != nil {
			return fmt.Errorf("fix failed: %w", err)
		}

		if results.Fixed == 0 && results.Added == 0 {
			fmt.Println("✓ No files needed fixing")
		} else {
			if results.Fixed > 0 {
				fmt.Printf("✓ Fixed %d files\n", results.Fixed)
			}
			if results.Added > 0 {
				fmt.Printf("✓ Added headers to %d files\n", results.Added)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)
}
