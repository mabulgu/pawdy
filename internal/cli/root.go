// Package cli provides the command-line interface for Pawdy.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	safety  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pawdy",
	Short: "Your bare-metal onboarding assistant",
	Long: `Pawdy is a production-ready, fully local command-line chat assistant 
designed to help engineers onboard to the OpenShift Bare Metal team. 
It runs entirely offline using Meta's Llama models and provides RAG 
(Retrieval-Augmented Generation) capabilities over your team documentation.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./pawdy.yaml)")
	rootCmd.PersistentFlags().StringVar(&safety, "safety", "", "safety mode (on|off)")
	
	// Bind flags to viper
	viper.BindPFlag("safety", rootCmd.PersistentFlags().Lookup("safety"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory and standard locations
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.pawdy")
		viper.AddConfigPath("/etc/pawdy")
		viper.SetConfigName("pawdy")
		viper.SetConfigType("yaml")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("PAWDY")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}
