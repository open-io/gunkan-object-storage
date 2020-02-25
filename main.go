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
	"github.com/jfsmig/object-storage/internal/cmd-blob-server"
	"github.com/jfsmig/object-storage/internal/cmd-kv-client"
	"github.com/jfsmig/object-storage/internal/cmd-kv-server"
	"github.com/spf13/cobra"

	"errors"
	"log"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gunkan",
		Short: "Manage your data and metadata on hunkan services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Missing subcommand")
		},
	}

	blobCmd := &cobra.Command{
		Use:   "blob",
		Short: "BLOB related tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Missing subcommand")
		},
	}

	kvCmd := &cobra.Command{
		Use:   "kv",
		Short: "KV related tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Missing subcommand")
		},
	}

	blobCmd.AddCommand(cmd_blob_server.MainCommand(), cmd_blob_client.MainCommand())
	kvCmd.AddCommand(cmd_kv_server.MainCommand(), cmd_kv_client.MainCommand())
	rootCmd.AddCommand(blobCmd, kvCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln("Command error:", err)
	}
}
