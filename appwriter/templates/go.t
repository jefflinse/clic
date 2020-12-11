package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "{{.Name}}",
}

func init() {
{{range .Commands}}
	rootCmd.AddCommand(&cobra.Command{
		Use:   "{{.Name}}",
		Short: "{{.Description}}",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hello, world!")
		},
	})
{{end}}
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	execute()
}
