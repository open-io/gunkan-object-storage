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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"fmt"
)

func GetCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"fetch", "retrieve"},
		Short:   "Get a value from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_kv_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			// Unpack the positional arguments
			if len(args) < 2 {
				return errors.New("Missing BASE or KEY")
			}
			base := args[0]

			for _, key := range args[1:] {
				value, err := client.Get(cmd.Context(), base, key)
				if err != nil {
					return err
				}
				fmt.Printf("%s %s", key, value)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
