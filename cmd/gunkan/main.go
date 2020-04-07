//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

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
