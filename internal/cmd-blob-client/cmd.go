//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_client

import (
	"github.com/spf13/cobra"

	"errors"
	"log"
)

func MainCommand() *cobra.Command {
	client := &cobra.Command{
		Use:     "cli",
		Aliases: []string{"client"},
		Short:   "Client of BLOB services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Missing subcommand")
		},
	}
	client.AddCommand(PutCommand())
	client.AddCommand(GetCommand())
	client.AddCommand(DelCommand())
	client.AddCommand(ListCommand())
	client.AddCommand(StatusCommand())
	client.AddCommand(HealthCommand())
	return client
}

func debug(id string, err error) {
	if err == nil {
		log.Printf("%v OK", id)
	} else {
		log.Printf("%v %v", id, err)
	}
}

// Common configuration for all the subcommands
type config struct {
	url string
}
