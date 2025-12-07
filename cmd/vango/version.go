package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print version, commit, and build information for the Vango CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(version)
				return
			}

			printBanner()
			fmt.Println()
			fmt.Printf("  Version:    %s\n", version)
			fmt.Printf("  Commit:     %s\n", commit)
			fmt.Printf("  Built:      %s\n", date)
			fmt.Printf("  Go version: %s\n", runtime.Version())
			fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Println()
		},
	}

	cmd.Flags().BoolVarP(&short, "short", "s", false, "Print only version number")

	return cmd
}
