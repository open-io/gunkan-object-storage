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
	"github.com/jfsmig/object-storage/pkg/blob-client"
	"github.com/jfsmig/object-storage/pkg/blob-model"
	"github.com/spf13/cobra"

	"errors"
	"flag"
	"fmt"
)

func ListCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"list"},
		Short:   "List items stored on a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var id gunkan_blob_model.Id

			if flag.NArg() > 1 {
				if flag.NArg() > 2 {
					return errors.New("Too many BLOB id")
				}
				err = id.Decode(flag.Arg(1))
				if err != nil {
					return err
				}
			}

			client, err := gunkan_blob_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			var items []gunkan_blob_model.Id
			if len(id.Content) <= 0 {
				items, err = client.List(1000)
			} else {
				items, err = client.ListAfter(1000, id)
			}
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Println(item.Encode())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
