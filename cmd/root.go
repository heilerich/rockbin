package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string
var version = "v0.2.1"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rockbin",
	Short: "Send bin capacity to a mqtt server",
	Long: `This app is designed to let you periodically send 
the bin capacity (as time or a percentage) to a mqtt server.`,
	Version: version,
	Run:     func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())

}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/mnt/data/rockbin/rockbin.yaml", "config file")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("yaml")

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Info("no config file")
	}
}
