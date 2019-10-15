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
	"io"
	"log"
	"os"
)

type getCmd struct{}

func (*getCmd) Name() string     { return "get" }
func (*getCmd) Synopsis() string { return "Get a BLOB from the store." }
func (*getCmd) Usage() string {
	return `
get BLOB_ID

`
}

func (p *getCmd) SetFlags(f *flag.FlagSet) {}

func (p *getCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var id blob.Id

	if flag.NArg() != 2 {
		log.Println("Missing Blob ID")
		return subcommands.ExitFailure
	}

	if err = id.Decode(flag.Arg(1)); err != nil {
		log.Printf("Blob ID parsing error: %v", err)
		return subcommands.ExitFailure
	}

	var r io.ReadCloser
	if r, err = client.Get(id); err != nil {
		log.Printf("GET(%v) error: %v", id, err)
		return subcommands.ExitFailure
	} else {
		_, err = io.Copy(os.Stdout, r)
		r.Close()
		if err != nil {
			log.Printf("GET error: %v", err)
			return subcommands.ExitFailure
		} else {
			return subcommands.ExitSuccess
		}
	}
}
