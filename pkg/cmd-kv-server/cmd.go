//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_server

import (
	"github.com/spf13/cobra"

	"errors"
)

type config struct {
	addrAnnounce string
	addrBind string
}

func MainCommand() *cobra.Command {
	var cfg config

	server := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server", "service", "worker"},
		Short:   "Start a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("NYI")
		},
	}

	server.Flags().StringVar(&cfg.addrAnnounce, "me", "", "Specify a different address than the bind address")

	return server
}

