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
	"fmt"
	"github.com/jfsmig/object-storage/pkg/kv-client"
	"github.com/spf13/cobra"
)

func HealthCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "ping",
		Aliases: []string{"ls"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_kv_client.Dial(cfg.url)
			if err != nil {
				return err
			}
			state, err := client.Health(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(state)
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
