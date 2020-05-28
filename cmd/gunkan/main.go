// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/jfsmig/object-storage/internal/cmd-blob-client"
	"github.com/jfsmig/object-storage/internal/cmd-index-client"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gunkan",
		Short: "Manage your data and metadata on hunkan services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cobra.ErrSubCommandRequired
		},
	}
	gunkan.PatchCommandLogs(rootCmd)

	blobCmd := cmd_blob_client.MainCommand()
	blobCmd.Use = "blob"
	blobCmd.Aliases = []string{}

	kvCmd := cmd_index_client.MainCommand()
	kvCmd.Use = "kv"
	kvCmd.Aliases = []string{}

	rootCmd.AddCommand(blobCmd)
	rootCmd.AddCommand(kvCmd)
	if err := rootCmd.Execute(); err != nil {
		gunkan.Logger.Fatal().Err(err).Msg("Command error")
	}
}
