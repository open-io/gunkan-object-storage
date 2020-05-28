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
)

func DelCommand() *cobra.Command {
	var cfg config

	client := &cobra.Command{
		Use:     "del",
		Aliases: []string{"delete", "remove", "rm", "erase"},
		Short:   "Delete BLOBs from a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New(("Missing Blob ID"))
			}
			client, err := gunkan.DialBlob(cfg.url)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				id := args[0]
				err = delOne(client, id)
				debug(id, err)
				return err
			}

			strongError := false
			for _, id := range args {
				err = delOne(client, id)
				debug(id, err)
				if err != gunkan.ErrNotFound {
					strongError = true
				}
			}
			if strongError {
				err = errors.New("Unrecoverable error")
			} else {
				err = nil
			}
			return err
		},
	}

	client.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return client
}

func delOne(client gunkan.BlobClient, strid string) error {
	var err error

	if _, err = gunkan.DecodeBlobId(strid); err != nil {
		return err
	} else {
		return client.Delete(context.Background(), strid)
	}
}
