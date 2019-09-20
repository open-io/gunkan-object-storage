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

	"fmt"
	"context"
	"flag"
	"github.com/google/subcommands"
	"log"
)

type listCmd struct{}

func (*listCmd) Name() string     { return "ls" }
func (*listCmd) Synopsis() string { return "List the keys in the store" }
func (*listCmd) Usage() string {
	return `
list BASE [MARKER]
`
}

func (p *listCmd) SetFlags(f *flag.FlagSet) {}

func (p *listCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var client kv.Client

	if len(targets) <= 0 {
		log.Printf("No target specified")
		return subcommands.ExitUsageError
	}

	// Unpack the positional arguments
	if flag.NArg() < 2 {
		log.Printf("Missing BASE")
		return subcommands.ExitUsageError
	}
	if flag.NArg() > 3 {
		log.Printf("Too many MARKER")
		return subcommands.ExitUsageError
	}
	var base, marker string
	base = flag.Arg(1)
	if flag.NArg() == 3 {
		marker = flag.Arg(2)
	}

	// Prepare the connection
	client, err = kv.Dial(targets)
	if err != nil {
		log.Printf("Client connection error: %v", err)
		return subcommands.ExitFailure
	}

	items, err := client.List(base, marker)
	if err != nil {
		log.Printf("LIST error: %v", err)
		return subcommands.ExitFailure
	}

	for _, item := range items {
		fmt.Println(item.Key, item.Version)
	}
	return subcommands.ExitSuccess
}
