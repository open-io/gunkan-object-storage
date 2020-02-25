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
	"github.com/jfsmig/object-storage/pkg/blob-model"
	"github.com/spf13/cobra"

	"errors"
	"io"
	"os"
)

func GetCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"fetch", "retrieve", "download", "dl"},
		Short:   "Get data from a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return errors.New("Missing Blob ID")
			}
			client, err := gunkan_blob_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			return getOne(client, args[0])
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

func getOne(client gunkan_blob_client.Client, strid string) error {
	var err error
	var id gunkan_blob_model.Id

	if err = id.Decode(strid); err != nil {
		return err
	} else {
		r, err := client.Get(id)
		if err != nil {
			return err
		} else {
			defer r.Close()
			io.Copy(os.Stdout, r)
			return nil
		}
	}
}
