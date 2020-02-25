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
	"encoding/json"
	"github.com/jfsmig/object-storage/pkg/kv-client"
	"github.com/spf13/cobra"
	"os"
)

func StatusCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stats", "stat"},
		Short:   "Get usage statistics from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_kv_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			st, err := client.Status(cmd.Context())
			if err == nil {
				json.NewEncoder(os.Stdout).Encode(&st)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
