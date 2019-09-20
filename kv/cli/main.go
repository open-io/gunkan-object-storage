//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package main

import (
	"context"
	"flag"
	"github.com/google/subcommands"
	"os"
)

var targets string

func main() {
	flag.StringVar(&targets, "target", os.Getenv("GUNKAN_KV_TARGET"),
		`A connection string compatible with nanomsg-ng.
By default, the GUNKAN_KV_TARGET environment is considered.`)

	subcommands.Register(subcommands.HelpCommand(), "Helpers")
	subcommands.Register(subcommands.FlagsCommand(), "Helpers")
	subcommands.Register(subcommands.CommandsCommand(), "Helpers")
	subcommands.Register(&floodCmd{}, "Actions")
	subcommands.Register(&pingCmd{}, "Actions")
	subcommands.Register(&putCmd{}, "Actions")
	subcommands.Register(&getCmd{}, "Actions")
	subcommands.Register(&listCmd{}, "Actions")

	flag.Parse()

	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
