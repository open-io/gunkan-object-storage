// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_index_store_rocksdb

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

	cmd := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server"},
		Short:   "Start an index server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("Missing positional args: ADDR DIRECTORY")
			} else {
				cfg.addrBind = args[0]
				cfg.dirBase = args[1]
			}

			// FIXME(jfsmig): Fix the sanitizing of the input
			if cfg.addrBind == "" {
				return errors.New("Missing bind address")
			}
			if cfg.dirBase == "" {
				return errors.New("Missing base directory")
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
	cmd.Flags().StringVar(&cfg.dirConfig, "tls", "", tlsUsage)
	cmd.Flags().StringVar(&cfg.addrAnnounce, "pub", "", publicUsage)
	return cmd
}
