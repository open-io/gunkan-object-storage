//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_index_gate

import (
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/internal/helpers-grpc"
	"github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"net"
)

func MainCommand() *cobra.Command {
	var cfg serviceConfig

	cmd := &cobra.Command{
		Use:     "gate",
		Aliases: []string{"proxy", "gateway"},
		Short:   "Start a stateless index gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("Missing positional args: ADDR")
			} else {
				cfg.AddrBind = args[0]
			}

			// FIXME(jfsmig): Fix the sanitizing of the input
			if cfg.AddrBind == "" {
				return errors.New("Missing bind address")
			}
			if cfg.AddrAnnounce == "" {
				cfg.AddrAnnounce = cfg.AddrBind
			}

			lis, err := net.Listen("tcp", cfg.AddrBind)
			if err != nil {
				return errors.New(fmt.Sprintf("Listen error (%s): %s", cfg.AddrBind, err.Error()))
			}
			service, err := NewService()
			if err != nil {
				return err
			}
			httpServer := helpers_grpc.ServerTLS(cfg.AddrAnnounce, func(srv *grpc.Server) {
				gunkan_index_proto.RegisterIndexServer(srv, service)
				gunkan_index_proto.RegisterDiscoveryServer(srv, service)
			})
			return httpServer.Serve(lis)
		},
	}

	const (
		publicUsage = "Public address of the service."
		configUsage = "Specify a different address than the bind address"
	)
	cmd.Flags().StringVar(&cfg.AddrAnnounce, "pub", "", publicUsage)
	cmd.Flags().StringVar(&cfg.DirConfig, "f", "", configUsage)
	return cmd
}
