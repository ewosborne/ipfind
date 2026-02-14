/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	matchExact   bool
	matchSubnet  bool
	matchLongest bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ipfind",
	Short: "Finds matching IP addresses",
	Long: `ipfind searches a file or stdin for matching ip addresses.  Input parameters
	is address with optional mask. Flag documentation TBD.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		fmt.Println("in prerun", matchExact, matchSubnet, matchLongest)
		// TODO check that no more than one is defined
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hi from run")
	},
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ipfind.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolVarP(&matchExact, "exact", "e", false, "Require exact match")
	rootCmd.Flags().BoolVarP(&matchSubnet, "subnet", "s", false, "Print all matching subnets")
	rootCmd.Flags().BoolVarP(&matchLongest, "longest", "l", true, "Print longest match")

}
