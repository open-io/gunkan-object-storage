//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_store_fs

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
)

func MainCommand() *cobra.Command {
	var flagSMR bool
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

			var err error
			var repo Repo
			if flagSMR {
				repo, err = MakePostNamed(cfg.dirBase)
			} else {
				repo, err = MakePreNamed(cfg.dirBase)
			}
			if err != nil {
				return errors.New(fmt.Sprintf("Repository error [%s]", cfg.dirBase, err.Error()))
			}

			srv := service{repo: repo, config: cfg}
			http.HandleFunc(routeInfo, wrap(&srv, get(handleInfo())))
			http.HandleFunc(routeStatus, wrap(&srv, get(handleStatus())))
			http.HandleFunc(routeHealth, wrap(&srv, get(handleHealth())))
			http.HandleFunc(prefixBlob, wrap(&srv, handleBlob()))
			http.HandleFunc(routeList, wrap(&srv, get(handleList())))
			err = http.ListenAndServe(cfg.addrBind, nil)
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
	server.Flags().BoolVar(&flagSMR, "smr", false, smrUsage)
	return server
}
