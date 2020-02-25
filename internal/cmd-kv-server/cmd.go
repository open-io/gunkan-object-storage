//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_server

import (
	"github.com/jfsmig/object-storage/pkg/kv-proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"errors"
	"fmt"
	"net"
)

func MainCommand() *cobra.Command {
	var addrBind string
	var cfg Config

	cmd := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server"},
		Short:   "Start a KV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("Missing positional args: ADDR DIRECTORY")
			}

			addrBind = args[0]
			cfg.BaseDir = args[1]
			if cfg.AddrAnnounce == "" {
				cfg.AddrAnnounce = addrBind
			}
			if addrBind == "" {
				return errors.New("Missing bind address")
			}
			if cfg.BaseDir == "" {
				return errors.New("Missing base directory")
			}

			lis, err := net.Listen("tcp", addrBind)
			if err != nil {
				return errors.New(fmt.Sprintf("Listen error (%s): %s", addrBind, err.Error()))
			}

			service, err := NewService(cfg)
			if err != nil {
				return errors.New(fmt.Sprintf("Repository error (%s): %s", cfg.BaseDir, err.Error()))
			}

			srv := grpc.NewServer()
			gunkan_kv_proto.RegisterKVServer(srv, service)
			return srv.Serve(lis)
		},
	}

	cmd.Flags().StringVar(&cfg.AddrAnnounce, "me", "", "Specify a different address than the bind address")

	return cmd
}
