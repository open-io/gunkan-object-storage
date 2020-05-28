// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_client

import (
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
)

func MainCommand() *cobra.Command {
	client := &cobra.Command{
		Use:     "cli",
		Aliases: []string{"client"},
		Short:   "Client of BLOB services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cobra.ErrSubCommandRequired
		},
	}
	client.AddCommand(PutCommand())
	client.AddCommand(GetCommand())
	client.AddCommand(DelCommand())
	client.AddCommand(ListCommand())
	client.AddCommand(SrvInfoCommand())
	client.AddCommand(SrvHealthCommand())
	client.AddCommand(SrvMetricsCommand())
	return client
}

func debug(id string, err error) {
	gunkan.Logger.Info().Str("id", id).Err(err)
	if err == nil {
		gunkan.Logger.Info().Str("id", id).Msg("ok")
	} else {
		gunkan.Logger.Info().Str("id", id).Err(err)
	}
}

// Common configuration for all the subcommands
type config struct {
	url string
}
