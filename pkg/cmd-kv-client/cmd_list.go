//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_client

import (
	"github.com/jfsmig/object-storage/pkg/kv-client"
	"github.com/spf13/cobra"

	"errors"
	"fmt"
)

func ListCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Get a slice of keys from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_kv_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			if len(args) < 1 {
				return errors.New("Missing BASE")
			}
			if len(args) > 3 {
				return errors.New("Too many MARKER")
			}
			var base, marker string
			base = args[0]
			if len(args) == 2 {
				marker = args[1]
			}

			items, err := client.List(cmd.Context(), base, marker)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Println(item.Key, item.Version)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
