// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_store_fs

import (
	"errors"
	"fmt"
	ghttp "github.com/jfsmig/object-storage/internal/helpers-http"
	"github.com/spf13/cobra"
	"net/http"
)

func MainCommand() *cobra.Command {
	var cfg config

	server := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server", "service", "worker", "agent"},
		Short:   "Start a BLOB server",
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

			srv, err := newService(cfg)
			if err != nil {
				return errors.New(fmt.Sprintf("Repository error [%s]", cfg.dirBase, err.Error()))
			}

			api := ghttp.NewHttpApi(cfg.addrAnnounce, infoString)
			api.Route(routeList, ghttp.Get(srv.handleList()))
			api.Route(prefixData, srv.handleBlob())
			err = http.ListenAndServe(cfg.addrBind, api.Handler())
			if err != nil {
				return errors.New(fmt.Sprintf("HTTP error [%s]", cfg.addrBind, err.Error()))
			}
			return nil
		},
	}

	const (
		publicUsage = "Public address of the service"
		tlsUsage    = "Path to a directory with the TLS configuration"
		smrUsage    = "Use a SMR ready naming policy of objects"
	)
	server.Flags().StringVar(&cfg.dirConfig, "tls", "", tlsUsage)
	server.Flags().StringVar(&cfg.addrAnnounce, "pub", "", publicUsage)
	return server
}
