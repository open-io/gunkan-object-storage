// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/jfsmig/object-storage/internal/cmd-index-gate"
	"github.com/jfsmig/object-storage/pkg/gunkan"
)

func main() {
	rootCmd := cmd_index_gate.MainCommand()
	gunkan.PatchCommandLogs(rootCmd)
	rootCmd.Use = "gunkan-index-gate"
	if err := rootCmd.Execute(); err != nil {
		gunkan.Logger.Fatal().Err(err).Msg("Command error")
	}
}
