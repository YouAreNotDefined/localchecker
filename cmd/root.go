package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

type CfgMap struct {
	K string `toml:"K"`
	V string `toml:"V"`
}

type Config struct {
	Port      string   `toml:"Port"`
	Path      []CfgMap `toml:"Path"`
	IncludeId []CfgMap `toml:"IncludeId"`
	Alternate []CfgMap `toml:"Alternate,omitempty"`
}

var config Config

var rootCmd = &cobra.Command{
	Use:   "localchecker",
	Short: "Set up an http server",
	Long:  `This application is a tool to set up an http server`,
	Run: func(c *cobra.Command, args []string) {
		fmt.Printf("configFile: %s\nconfig: %#v\n", cfgFile, config)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "localchecker.toml", "config file")
}

func initConfig() {
	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
