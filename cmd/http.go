package main

import "github.com/spf13/cobra"

func run() error {
	cmd := &cobra.Command{
		Use:   "pagesy",
		Short: "reading(put a better description here)",
	}

	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}
