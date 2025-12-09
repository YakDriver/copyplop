// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/YakDriver/copyplop/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "copyplop",
	Short: "Manage copyright headers across codebases",
	Long: `copyplop is a configurable tool for managing copyright headers in source code files.
It can check for missing headers, fix incorrect ones, and handle any copyright format.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .copyplop.yaml)")
	rootCmd.PersistentFlags().StringP("path", "p", ".", "path to process")

	// Bind flags to viper
	viper.BindPFlag("path", rootCmd.PersistentFlags().Lookup("path"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".copyplop")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("COPYPLOP")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: Could not read config file: %v\n", err)
		os.Exit(1)
	}

	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}
}
