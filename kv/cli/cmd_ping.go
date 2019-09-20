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

type pingCmd struct{}

func (*pingCmd) Name() string     { return "ping" }
func (*pingCmd) Synopsis() string { return "Ping a KV service" }
func (*pingCmd) Usage() string    { return `ping` }

func (p *pingCmd) SetFlags(f *flag.FlagSet) {}

func (p *pingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(targets) <= 0 {
		log.Printf("No target specified")
		return subcommands.ExitUsageError
	}

	client, err := kv.Dial(targets)
	if err != nil {
		log.Printf("Client connection error: %v", err)
		panic("CNX error")
	}

	if err = client.Ping(); err != nil {
		log.Printf("PING error: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
