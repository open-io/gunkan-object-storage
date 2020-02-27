//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blobindex_client

import (
	"encoding/json"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/blobindex-client"
	"github.com/spf13/cobra"
	"os"

	"errors"
)

func MainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cli",
		Aliases: []string{"client"},
		Short:   "Query a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Command not implemented")
		},
	}
	cmd.AddCommand(PutCommand())
	cmd.AddCommand(GetCommand())
	cmd.AddCommand(DeleteCommand())
	cmd.AddCommand(ListCommand())
	cmd.AddCommand(StatusCommand())
	cmd.AddCommand(HealthCommand())
	return cmd
}

type config struct {
	url string
}

func StatusCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stats", "stat"},
		Short:   "Get usage statistics from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
			if err != nil {
				return err
			}

			st, err := client.Status(cmd.Context())
			if err == nil {
				json.NewEncoder(os.Stdout).Encode(&st)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

func PutCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "put",
		Aliases: []string{"set"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
			if err != nil {
				return err
			}
			if len(args) != 3 {
				return errors.New("Missing BASE, KEY or VALUE")
			}
			base := args[0]
			key := args[1]
			value := args[2]

			return client.Put(cmd.Context(), base, key, value)
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

func ListCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Get a slice of keys from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
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

func HealthCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "ping",
		Aliases: []string{"ls"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
			if err != nil {
				return err
			}
			state, err := client.Health(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(state)
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}

func GetCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"fetch", "retrieve"},
		Short:   "Get a value from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
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

func DeleteCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "del",
		Aliases: []string{"delete", "remove", "erase", "rm"},
		Short:   "Delete an entry from a KV service",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gunkan_blobindex_client.Dial(cfg.url)
			if err != nil {
				return err
			}
			if len(args) != 2 {
				return errors.New("Missing BASE and/or KEY")
			}
			base := args[0]
			key := args[1]

			return client.Delete(cmd.Context(), base, key)
		},
	}

	cmd.Flags().StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")

	return cmd
}
