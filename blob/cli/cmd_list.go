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
	"fmt"
	"github.com/google/subcommands"
	"log"
)

type listCmd struct{}

func (*listCmd) Name() string     { return "ls" }
func (*listCmd) Synopsis() string { return "List the ID of the BLOB in the store" }
func (*listCmd) Usage() string {
	return `
ls [BLOB_ID]

`
}

func (p *listCmd) SetFlags(f *flag.FlagSet) {}

func (p *listCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var id blob.Id

	if flag.NArg() > 1 {
		if flag.NArg() > 2 {
			log.Println("Too many BLOB id")
			return subcommands.ExitFailure
		} else if err = id.Decode(flag.Arg(1)); err != nil {
			log.Printf("Blob ID parsing error: %v", err)
			return subcommands.ExitFailure
		}
	}
	log.Println(flag.NArg(), id)

	var items []blob.Id
	if len(id.Content) <= 0 {
		items, err = client.List(1000)
	} else {
		items, err = client.ListAfter(1000, id)
	}
	if err != nil {
		log.Printf("LIST error: %v", err)
		return subcommands.ExitFailure
	}

	for _, item := range items {
		fmt.Println(item.Encode())
	}
	return subcommands.ExitSuccess
}
