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
	"github.com/jfsmig/object-storage/internal/cmd-kv-server"

	"log"
)

func main() {
	rootCmd := cmd_kv_server.MainCommand()
	rootCmd.Use = "gunkan-kv-rocksdb"
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln("Command error:", err)
	}
}
