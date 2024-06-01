package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "budget",
	Short: "Budget is a minimal budgeting app",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello World!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(CreateCmd)
	rootCmd.AddCommand(AddCmd)
	rootCmd.AddCommand(StatusCmd)
}
