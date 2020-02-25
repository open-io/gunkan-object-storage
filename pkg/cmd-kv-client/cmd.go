//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_client

import (
	"github.com/spf13/cobra"

	"errors"
)

func MainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "client",
		Aliases: []string{"cli"},
		Short:   "Query a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Command not implemented")
		},
	}
	cmd.AddCommand(PutCommand())
	cmd.AddCommand(GetCommand())
	cmd.AddCommand(DeleteCommand())
	cmd.AddCommand(ListCommand())
	cmd.AddCommand(StatusCommand())
	cmd.AddCommand(HealthCommand())
	return cmd
}

type config struct {
	url string
}