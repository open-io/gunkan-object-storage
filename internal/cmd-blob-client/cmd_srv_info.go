// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
