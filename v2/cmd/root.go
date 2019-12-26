package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/Defacto2/uuid/v2/lib/logs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const configFilename = ".df2.yaml"

var (
	// Quiet disables most printing or output to terminal
	quiet    bool = false
	cfgFile  string
	home, _  = os.UserHomeDir()
	filepath = path.Join(home, configFilename)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uuid",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s)", configFilename))
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable most feedback to terminal")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	initQuiet(quiet)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(home)
		viper.SetConfigName(configFilename)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && !quiet {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// initQuiet quiets the terminal output
func initQuiet(q bool) {
	logs.Quiet = q
}
