// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var (
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	flagVerbose = 0
	flagQuiet   = false
)

func PatchCommandLogs(cmd *cobra.Command) {
	cmd.PersistentFlags().CountVarP(&flagVerbose, "verbose", "v", "Increase the verbosity level")
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", flagQuiet, "Shut the logs")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if flagQuiet {
			zerolog.SetGlobalLevel(zerolog.Disabled)
		} else {
			switch flagVerbose {
			case 0:
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			case 1:
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			case 2:
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
			}
		}
	}
}
