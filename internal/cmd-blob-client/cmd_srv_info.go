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
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
	"os"
)

func SrvInfoCommand() *cobra.Command {
	var cfg config

	client := &cobra.Command{
		Use:     "info",
		Aliases: []string{"describe", "wot", "who"},
		Short:   "Get the service type",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}
			body, err := client.(gunkan.HttpMonitorClient).Info(context.Background())
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(body)
			return err
		},
	}

	client.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return client
}
