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
	"os"

	"errors"
)

func PutCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "put",
		Aliases: []string{"push", "store", "add"},
		Short:   "Put data in a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var id gunkan_blob_model.Id

			if len(args) < 1 {
				return errors.New("Missing Blob ID")
			}
			if err = id.Decode(args[0]); err != nil {
				return err
			}

			client, err := gunkan_blob_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			if len(args) == 2 {
				path := args[1]
				if fin, err := os.Open(path); err == nil {
					defer fin.Close()
					var finfo os.FileInfo
					finfo, err = fin.Stat()
					if err == nil {
						err = client.PutN(id, fin, finfo.Size())
					}
				}
			} else {
				err = client.Put(id, os.Stdin)
			}

			return err
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

