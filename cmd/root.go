package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/frankstratton/docdr/docdr"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "docdr",
		Short: "DocDr is a tool to help you add godoc comments to packages.",
		Long: `A CLI tool to help find missing godoc comments and easily 
add new ones.  See https://github.com/frankstratton/DocDr for more details.`,
	}

	runCmd = &cobra.Command{
		Use:   "run [directory] [package]",
		Short: "Run DocDr on a package",
		Long:  ``,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires a target directory to scan with optional package")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var pack = ""

			if len(args) > 1 {
				pack = args[1]
			}

			docdr.ScanPackage(args[0], pack)
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)
}

// Run the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
