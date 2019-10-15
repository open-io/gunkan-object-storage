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
	blob "../client/golang"

	"context"
	"flag"
	"github.com/google/subcommands"
	"log"
	"os"
)

var client blob.Client

func main() {
	var targets string

	flag.StringVar(&targets, "target", os.Getenv("GUNKAN_BLOB_TARGET"),
		`A IP:PORT endpoint to a BLOB service.
By default, the GUNKAN_BLOB_TARGET environment is considered.`)

	subcommands.Register(subcommands.HelpCommand(), "Helpers")
	subcommands.Register(subcommands.FlagsCommand(), "Helpers")
	subcommands.Register(subcommands.CommandsCommand(), "Helpers")
	subcommands.Register(&putCmd{}, "Actions")
	subcommands.Register(&getCmd{}, "Actions")
	subcommands.Register(&delCmd{}, "Actions")
	subcommands.Register(&listCmd{}, "Actions")
	subcommands.Register(&statusCmd{}, "Actions")

	flag.Parse()

	ctx := context.Background()

	// Prepare the connection
	var err error
	if len(targets) <= 0 {
		log.Fatalf("No target specified")
	}
	client, err = blob.Dial(targets)
	if err != nil {
		log.Fatalf("Client connection error: %v", err)
	}

	os.Exit(int(subcommands.Execute(ctx)))
}
