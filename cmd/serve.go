package cmd

import (
	"github.com/YouAreNotDefined/localchecker/internal/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start up the http server. File names can be omitted.",
	Long:  `Start up the http server. File names can be omitted.`,
	Run:   serve.Listen,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
