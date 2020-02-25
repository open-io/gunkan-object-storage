//
// Copyright 2019-2020 Jean-Francois Smigielski
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
	var flagSMR bool
	var addrBind string
	var baseDir string
	var cfg config

	server := &cobra.Command{
		Use:     "srv",
		Aliases: []string{"server", "service", "worker", "agent"},
		Short:   "Start a BLOB server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := service{repo: nil, config: cfg}

			if len(args) != 2 {
				return errors.New("Missing positional args: ADDR DIRECTORY")
			}
			addrBind = args[0]
			baseDir = args[1]
			if cfg.addrAnnounce == "" {
				cfg.addrAnnounce = addrBind
			}
			if addrBind == "" {
				return errors.New("Missing bind address")
			}
			if baseDir == "" {
				return errors.New("Missing base directory")
			}

			var err error
			if flagSMR {
				srv.repo, err = MakePostNamed(baseDir)
			} else {
				srv.repo, err = MakePreNamed(baseDir)
			}
			if err != nil {
				return errors.New(fmt.Sprintf("Repository error [%s]", baseDir, err.Error()))
			}

			http.HandleFunc(prefixBlob, wrap(&srv, handleBlob()))
			http.HandleFunc("/v1/list", wrap(&srv, get(handleList())))
			http.HandleFunc("/v1/info", wrap(&srv, get(handleInfo())))
			http.HandleFunc("/v1/status", wrap(&srv, get(handleStatus())))
			http.HandleFunc("/v1/health", wrap(&srv, get(handleHealth())))
			err = http.ListenAndServe(addrBind, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("HTTP error [%s]", addrBind, err.Error()))
			}
			return nil
		},
	}

	server.Flags().BoolVar(&flagSMR, "smr", false, "Use SMR ready naming")
	server.Flags().StringVar(&cfg.addrAnnounce, "me", "", "Specify a different address than the bind address")

	return server
}
