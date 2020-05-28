// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_index_gate

import (
	"errors"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jfsmig/object-storage/internal/helpers-grpc"
	"github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"net"
	"net/http"
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
				return err
			}
			service, err := NewService(cfg)
			if err != nil {
				return err
			}
			httpServer, err := helpers_grpc.ServerTLS(cfg.dirConfig)
			if err != nil {
				return err
			}

			gunkan_index_proto.RegisterIndexServer(httpServer, service)
			grpc_prometheus.Register(httpServer)
			http.Handle("/metrics", promhttp.Handler())
			http.HandleFunc("/info", func(rep http.ResponseWriter, req *http.Request) {
				rep.Write([]byte("Yallah!"))
			})
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
