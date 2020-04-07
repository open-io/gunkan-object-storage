//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

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
