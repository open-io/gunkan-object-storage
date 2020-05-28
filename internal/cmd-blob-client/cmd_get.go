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
			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}

			return getOne(client, args[0])
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

func getOne(client gunkan.BlobClient, strid string) error {
	var err error

	if _, err = gunkan.DecodeBlobId(strid); err != nil {
		return err
	} else {
		r, err := client.Get(context.Background(), strid)
		if err != nil {
			return err
		} else {
			defer r.Close()
			io.Copy(os.Stdout, r)
			return nil
		}
	}
}
