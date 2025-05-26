package cmd

import (
	"context"
	"github.com/spf13/cobra"
)

func Run() error {
	ctx := context.Background()

	cmd := &cobra.Command{
		Use:   "pagesy",
		Short: "reading(put a better description here)",
	}

	cmd.AddCommand(HTTPCommand(ctx))

	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}
