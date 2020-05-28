// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_client

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
)

func ListCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"list"},
		Short:   "List items stored on a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var realid string

			if flag.NArg() > 1 {
				if flag.NArg() > 2 {
					return errors.New("Too many BLOB id")
				}
				realid = flag.Arg(1)
			}

			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}

			var items []gunkan.BlobListItem
			if len(realid) <= 0 {
				items, err = client.List(context.Background(), 1000)
			} else {
				items, err = client.ListAfter(context.Background(), 1000, realid)
			}
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Println(item)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
