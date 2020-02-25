//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_server

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
)

func MainCommand() *cobra.Command {
	var cfg config

	server := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server"},
		Short:   "Start a BLOB server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := service{repo: nil, config: cfg}

			if len(args) != 2 {
				return errors.New("Missing positional args: ADDR DIRECTORY")
			}
			cfg.addrLocal = args[0]
			cfg.baseDir = args[1]
			if cfg.addrAnnounce == "" {
				cfg.addrAnnounce = cfg.addrLocal
			}
			if cfg.addrLocal == "" {
				return errors.New("Missing bind address")
			}
			if cfg.baseDir == "" {
				return errors.New("Missing base directory")
			}

			var err error
			if cfg.flagSMR {
				srv.repo, err = MakePostNamed(cfg.baseDir)
			} else {
				srv.repo, err = MakePreNamed(cfg.baseDir)
			}
			if err != nil {
				return errors.New(fmt.Sprintf("Repository error [%s]", cfg.baseDir, err.Error()))
			}

			http.HandleFunc(prefixBlob, wrap(&srv, handleBlob()))
			http.HandleFunc("/v1/list", wrap(&srv, get(handleList())))
			http.HandleFunc("/v1/info", wrap(&srv, get(handleInfo())))
			http.HandleFunc("/v1/status", wrap(&srv, get(handleStatus())))
			http.HandleFunc("/v1/health", wrap(&srv, get(handleHealth())))
			err = http.ListenAndServe(cfg.addrLocal, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("HTTP error [%s]", cfg.addrLocal, err.Error()))
			}
			return nil
		},
	}

	server.Flags().BoolVar(&cfg.flagSMR, "smr", false, "Use SMR ready naming")
	server.Flags().StringVar(&cfg.addrAnnounce, "me", "", "Specify a different address than the bind address")

	return server
}
