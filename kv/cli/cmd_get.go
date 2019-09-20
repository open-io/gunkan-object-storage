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
	"fmt"
	"github.com/google/subcommands"
	"log"
)

type getCmd struct{}

func (*getCmd) Name() string     { return "get" }
func (*getCmd) Synopsis() string { return "Get a value from the store." }
func (*getCmd) Usage() string {
	return `
get BASE KEY [KEY...]
`
}

func (p *getCmd) SetFlags(f *flag.FlagSet) {}

func (p *getCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var client kv.Client

	if len(targets) <= 0 {
		log.Printf("No target specified")
		return subcommands.ExitUsageError
	}

	// Unpack the positional arguments
	if flag.NArg() < 3 {
		log.Println("Missing BASE or KEY")
		return subcommands.ExitUsageError
	}
	base := flag.Arg(1)

	// Prepare the connection
	client, err = kv.Dial(targets)
	if err != nil {
		log.Printf("Client connection error: %v", err)
		return subcommands.ExitFailure
	}

	rc := subcommands.ExitSuccess

	for i := 2; i < flag.NArg(); i++ {
		key := flag.Arg(i)

		var value []byte
		value, err = client.Get(base, key)
		if err != nil {
			log.Printf("GET error: %v", err)
			fmt.Printf("%s %v", key, '-')
			rc = subcommands.ExitFailure
		} else {
			fmt.Printf("%s %s", key, string(value))
		}
	}

	return rc
}
