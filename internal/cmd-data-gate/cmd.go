//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_data_gate

import (
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/spf13/cobra"
	"net/http"
)

const (
	routeInfo   = "/info"
	routeHealth = "/health"
	routeStatus = "/status"
	routeList   = "/v1/list"
	prefixBlob  = "/v1/part/"
	infoString  = "gunkan/data-gate-" + gunkan.VersionString
)

func MainCommand() *cobra.Command {
	var cfg config

	server := &cobra.Command{
		Use:     "proxy",
		Aliases: []string{},
		Short:   "Start a BLOB proxy",
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

			srv, err := NewService(cfg)
			if err != nil {
				return err
			}
			http.HandleFunc(routeInfo, wrap(srv, get(handleInfo())))
			http.HandleFunc(routeStatus, wrap(srv, get(handleStatus())))
			http.HandleFunc(routeHealth, wrap(srv, get(handleHealth())))
			http.HandleFunc(prefixBlob, wrap(srv, handleBlob()))
			http.HandleFunc(routeList, wrap(srv, get(handleList())))
			err = http.ListenAndServe(cfg.addrBind, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("HTTP error [%s]", cfg.addrBind, err.Error()))
			}
			return nil
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
