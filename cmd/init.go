package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "An example command",
	Run: func(cmd *cobra.Command, args []string) {
		// Access systemInfo within the command's Run function
		fmt.Println("Blueprints Format:", systemInfo.Blueprints.Format)
		fmt.Println("Blueprints Location:", systemInfo.Blueprints.Location)
		fmt.Println("Blueprints Order:", systemInfo.Blueprints.Order)

		// Access other fields of systemInfo as needed
		// ...
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)
}
