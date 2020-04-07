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
	"bufio"
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"os"
)

func MainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cli",
		Aliases: []string{"client"},
		Short:   "Query a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cobra.ErrSubCommandRequired
		},
	}
	cmd.AddCommand(PutCommand())
	cmd.AddCommand(GetCommand())
	cmd.AddCommand(DeleteCommand())
	cmd.AddCommand(ListCommand())
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

func (cfg *config) dial() (gunkan.IndexClient, error) {
	if cfg.url != "" {
		gunkan.Logger.Debug().Msg("Explicit Index service endpoint")
		return gunkan.DialIndexGrpc(cfg.url, cfg.dirConfig)
	} else {
		gunkan.Logger.Debug().Msg("Polling an Index gate service endpoint")
		return gunkan.DialIndexPooled(cfg.dirConfig)
	}
}

func PutCommand() *cobra.Command {
	var cfg config
	var flagStdIn bool

	cmd := &cobra.Command{
		Use:     "put",
		Aliases: []string{"set"},
		Short:   "Check a service is up",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cfg.dial()
			if err != nil {
				return err
			}
			if flagStdIn {
				r := bufio.NewReader(os.Stdin)
				var base, key, value string
				for {
					n, err := fmt.Fscanln(r, &base, &key, &value)
					if err == io.EOF {
						return nil
					}
					if err != nil {
						return err
					}
					if n != 3 {
						return errors.New("Invalid line")
					}

					k := gunkan.BK(base, key)
					err = client.Put(cmd.Context(), k, value)
					if err != nil {
						gunkan.Logger.Warn().
							Str("base", base).Str("key", key).Str("value", value).
							Err(err)
					}
				}
			} else {
				if len(args) != 3 {
					return errors.New("Missing BASE, KEY or VALUE")
				}
				key := gunkan.BK(args[0], args[1])
				value := args[2]
				return client.Put(cmd.Context(), key, value)
			}
		},
	}

	cfg.prepare(cmd.Flags())
	cmd.Flags().BoolVarP(&flagStdIn, "stdin", "i", flagStdIn, "Consume triples from stdin")
	return cmd
}

func ListCommand() *cobra.Command {
	var cfg config
	var maxItems uint32 = gunkan.ListHardMax
	var flagFull bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Get a slice of keys from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cfg.dial()
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

			for {
				key := gunkan.BK(base, marker)
				items, err := client.List(cmd.Context(), key, maxItems)
				if err != nil {
					return err
				}
				if len(items) <= 0 {
					break
				}
				for _, item := range items {
					fmt.Println(item)
				}
				if flagFull {
					marker = items[len(items)-1]
				} else {
					break
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagFull, "full", "f", flagFull, "Iterate to the end of the list")
	cmd.Flags().Uint32VarP(&maxItems, "max", "n", maxItems, "Hint on the number of items received")
	cfg.prepare(cmd.Flags())
	return cmd
}

func GetCommand() *cobra.Command {
	var cfg config

	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"fetch", "retrieve"},
		Short:   "Get a value from a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cfg.dial()
			if err != nil {
				return err
			}

			// Unpack the positional arguments
			if len(args) < 2 {
				return errors.New("Missing BASE or KEY")
			}
			base := args[0]

			for _, key := range args[1:] {
				value, err := client.Get(cmd.Context(), gunkan.BK(base, key))
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
			client, err := cfg.dial()
			if err != nil {
				return err
			}
			if len(args) != 2 {
				return errors.New("Missing BASE and/or KEY")
			}

			return client.Delete(cmd.Context(), gunkan.BK(args[0], args[1]))
		},
	}

	cfg.prepare(cmd.Flags())
	return cmd
}
