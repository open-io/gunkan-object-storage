//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_client

import (
	"github.com/jfsmig/object-storage/pkg/blob-client"
	"github.com/spf13/cobra"

	"encoding/json"
	"os"
)

func StatusCommand() *cobra.Command {
	var cfg config

	client := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stats", "stat"},
		Short:   "Get the usage statistics of a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blob_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			st, err := client.Status();
			if err != nil {
				return err
			} else {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(&st)
				return nil
			}
		},
	}

	client.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return client
}
