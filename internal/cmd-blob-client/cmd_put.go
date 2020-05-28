// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_client

import (
	"context"
	"errors"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
	"os"
)

func PutCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "put",
		Aliases: []string{"push", "store", "add"},
		Short:   "Put data in a BLOB service",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var id gunkan.BlobId

			if len(args) < 1 {
				return errors.New("Missing Blob ID")
			}
			if id, err = gunkan.DecodeBlobId(args[0]); err != nil {
				return err
			}

			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}

			var realid string
			if len(args) == 2 {
				path := args[1]
				if fin, err := os.Open(path); err == nil {
					defer fin.Close()
					var finfo os.FileInfo
					finfo, err = fin.Stat()
					if err == nil {
						realid, err = client.PutN(context.Background(), id, fin, finfo.Size())
					}
				}
			} else {
				realid, err = client.Put(context.Background(), id, os.Stdin)
			}

			if err != nil {
				gunkan.Logger.Info().Str("id", id.Encode()).Str("real", realid).Msg("ok")
			}
			return err
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
