//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_index_client

import (
	"encoding/json"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
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
	url       string
	dirConfig string
}

func (cfg *config) prepare(fs *pflag.FlagSet) {
	fs.StringVar(&cfg.url, "url", "", "IP:PORT endpoint of the service to contact")
	fs.StringVar(&cfg.dirConfig, "f", "", "IP:PORT endpoint of the service to contact")
}

func (cfg *config) getUrl() (string, error) {
	if cfg.url != "" {
		log.Println("Explicit Index service endpoint")
		return cfg.url, nil
	} else {
		log.Println("Polling an Index gate service endpoint")
		if discovery, err := gunkan.NewDiscovery(); err != nil {
			return "", err
		} else {
			return discovery.PollIndexGate()
		}
	}
}

func StatusCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stats", "stat"},
		Short:   "Get usage statistics from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
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

	cfg.prepare(cmd.Flags())
	return cmd
}

func PutCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "put",
		Aliases: []string{"set"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
			if err != nil {
				return err
			}
			if len(args) != 3 {
				return errors.New("Missing BASE, KEY or VALUE")
			}
			key := gunkan.BaseKeyLatest(args[0], args[1])
			value := args[2]
			return client.Put(cmd.Context(), key, value)
		},
	}

	cfg.prepare(cmd.Flags())
	return cmd
}

func ListCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Get a slice of keys from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
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
			key := gunkan.BaseKeyLatest(base, marker)

			items, err := client.List(cmd.Context(), key, 1000)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Println(item.Key, item.Version)
			}
			return nil
		},
	}

	cfg.prepare(cmd.Flags())
	return cmd
}

func HealthCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "ping",
		Aliases: []string{"ls"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
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
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
			if err != nil {
				return err
			}

			// Unpack the positional arguments
			if len(args) < 2 {
				return errors.New("Missing BASE or KEY")
			}
			base := args[0]

			for _, key := range args[1:] {
				value, err := client.Get(cmd.Context(), gunkan.BaseKeyLatest(base, key))
				if err != nil {
					return err
				}
				fmt.Printf("%s %s", key, value)
			}
			return nil
		},
	}

	cfg.prepare(cmd.Flags())
	return cmd
}

func DeleteCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "del",
		Aliases: []string{"delete", "remove", "erase", "rm"},
		Short:   "Delete an entry from a KV service",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := cfg.getUrl()
			if err != nil {
				return err
			}
			client, err := gunkan.DialIndex(url, cfg.dirConfig)
			if err != nil {
				return err
			}
			if len(args) != 2 {
				return errors.New("Missing BASE and/or KEY")
			}

			return client.Delete(cmd.Context(), gunkan.BaseKeyLatest(args[0], args[1]))
		},
	}

	cfg.prepare(cmd.Flags())
	return cmd
}
