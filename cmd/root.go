/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/taylormonacelli/goldbug"
)

var (
	cfgFile   string
	verbose   bool
	logFormat string
	region    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fragiledonkey",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("Hello from fragiledonkey!")
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	var err error

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fragiledonkey.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose mode")

	err = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	if err != nil {
		slog.Error("error binding verbose flag", "error", err)
		os.Exit(1)
	}

	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "json or text (default is text)")

	err = viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))
	if err != nil {
		slog.Error("error binding log-format flag", "error", err)
		os.Exit(1)
	}

	rootCmd.PersistentFlags().StringVar(&region, "region", "us-west-2", "AWS region")

	err = viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
	if err != nil {
		slog.Error("error binding region flag", "error", err)
		os.Exit(1)
	}

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	err = viper.BindPFlag("toggle", rootCmd.Flags().Lookup("toggle"))
	if err != nil {
		slog.Error("error binding toggle flag", "error", err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".fragiledonkey" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".fragiledonkey")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	logFormat = viper.GetString("log-format")
	verbose = viper.GetBool("verbose")
	region = viper.GetString("region")

	slog.Debug("using config file", "path", viper.ConfigFileUsed())
	slog.Debug("log-format", "value", logFormat)
	slog.Debug("log-format", "value", viper.GetString("log-format"))
	slog.Debug("region", "value", region)
	slog.Debug("region", "value", viper.GetString("region"))

	setupLogging()
}

func setupLogging() {
	if verbose || logFormat != "" {
		if logFormat == "json" {
			goldbug.SetDefaultLoggerJson(slog.LevelDebug)
		} else {
			goldbug.SetDefaultLoggerText(slog.LevelDebug)
		}

		slog.Debug("setup", "verbose", verbose)
	}
}
