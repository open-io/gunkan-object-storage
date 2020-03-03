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
	"context"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
)

func HealthCommand() *cobra.Command {
	var cfg config

	client := &cobra.Command{
		Use:     "ping",
		Aliases: []string{"health"},
		Short:   "Check the health of a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}
			state, err := client.Health(context.Background())
			if err != nil {
				return err
			}
			fmt.Print(state)
			return nil
		},
	}

	client.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return client
}
