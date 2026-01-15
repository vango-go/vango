package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "VangoUI placeholder",
		Long: `VangoUI is planned but not yet implemented.

This command is reserved for the future VangoUI installer.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("VangoUI not yet implemented")
		},
	}

	return cmd
}
