// hello.go — implements the `movie hello` command.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/version"
)

var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Print a greeting",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("👋 Hello from movie-cli!")
		fmt.Printf("   Running version: %s\n", version.Short())
	},
}
