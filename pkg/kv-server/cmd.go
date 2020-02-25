//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_server

import (
	"github.com/spf13/cobra"

	"errors"
)

func MainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server"},
		Short:   "Start a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("NYI")
		},
	}

	return cmd
}
