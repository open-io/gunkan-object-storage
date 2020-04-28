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
	ghttp "github.com/jfsmig/object-storage/internal/helpers-http"
	"github.com/spf13/cobra"
	"net/http"
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

			srv, err := newService(cfg)
			if err != nil {
				return err
			}
			httpService := ghttp.NewHttpApi(cfg.addrAnnounce, infoString)
			httpService.Route(routeList, ghttp.Get(srv.handleList()))
			httpService.Route(prefixData, srv.handlePart())
			err = http.ListenAndServe(cfg.addrBind, httpService.Handler())
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
