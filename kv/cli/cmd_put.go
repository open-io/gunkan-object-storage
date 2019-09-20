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
	kv "../client/golang"

	"context"
	"flag"
	"github.com/google/subcommands"
	"log"
)

type putCmd struct{}

func (*putCmd) Name() string     { return "put" }
func (*putCmd) Synopsis() string { return "Put a value into the store." }
func (*putCmd) Usage() string {
	return `
put BASE KEY [VALUE]
`
}

func (p *putCmd) SetFlags(f *flag.FlagSet) {}

func (p *putCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var client kv.Client

	if len(targets) <= 0 {
		log.Printf("No target specified")
		return subcommands.ExitUsageError
	}

	// Unpack the positional arguments
	if flag.NArg() != 4 {
		log.Printf("Missing BASE, KEY or VALUE")
		return subcommands.ExitUsageError
	}
	base := flag.Arg(1)
	key := flag.Arg(2)
	value := flag.Arg(3)

	// Prepare the connection
	client, err = kv.Dial(targets)
	if err != nil {
		log.Printf("Client connection error: %v", err)
		return subcommands.ExitFailure
	}

	err = client.Put(base, key, []byte(value))
	if err != nil {
		log.Printf("PUT error: %v", err)
		return subcommands.ExitFailure
	}

	log.Println("Done")
	return subcommands.ExitSuccess
}
