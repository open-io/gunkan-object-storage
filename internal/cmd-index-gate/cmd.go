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

	server := &cobra.Command{
		Use:     "gate",
		Aliases: []string{"proxy", "gateway"},
		Short:   "Start a stateless index gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("Missing positional args: ADDR")
			} else {
				cfg.addrBind = args[0]
			}

			// FIXME(jfsmig): Fix the sanitizing of the input
			if cfg.addrBind == "" {
				return errors.New("Missing bind address")
			}
			if cfg.addrAnnounce == "" {
				cfg.addrAnnounce = cfg.addrBind
			}

			lis, err := net.Listen("tcp", cfg.addrBind)
			if err != nil {
				return errors.New(fmt.Sprintf("Listen error (%s): %s", cfg.addrBind, err.Error()))
			}
			service, err := NewService(cfg)
			if err != nil {
				return err
			}
			httpServer, err := helpers_grpc.ServerTLS(cfg.dirConfig, func(srv *grpc.Server) {
				gunkan_index_proto.RegisterIndexServer(srv, service)
				gunkan_index_proto.RegisterDiscoveryServer(srv, service)
			})
			if err != nil {
				return err
			}
			return httpServer.Serve(lis)
		},
	}

	const (
		publicUsage = "Public address of the service."
		tlsUsage    = "Path to a directory with the TLS configuration"
	)
	server.Flags().StringVar(&cfg.dirConfig, "tls", "", tlsUsage)
	server.Flags().StringVar(&cfg.addrAnnounce, "pub", "", publicUsage)
	return server
}
