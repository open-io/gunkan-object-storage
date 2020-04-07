//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package main

import (
	"github.com/jfsmig/object-storage/internal/cmd-data-gate"
	"github.com/jfsmig/object-storage/pkg/gunkan"
)

func main() {
	rootCmd := cmd_data_gate.MainCommand()
	gunkan.PatchCommandLogs(rootCmd)
	rootCmd.Use = "gunkan-data"
	if err := rootCmd.Execute(); err != nil {
		gunkan.Logger.Fatal().Err(err).Msg("Command error")
	}
}
