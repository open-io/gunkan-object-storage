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

type putCmd struct{}

func (*putCmd) Name() string     { return "put" }
func (*putCmd) Synopsis() string { return "Put a BLOB into the store." }
func (*putCmd) Usage() string {
	return `
put BLOB_ID [PATH_TO_BLOB]

`
}

func (p *putCmd) SetFlags(f *flag.FlagSet) {}

func (p *putCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	var id blob.Id

	if flag.NArg() < 2 {
		log.Println("Missing Blob ID")
		return subcommands.ExitFailure
	}
	if flag.NArg() > 3 {
		log.Println("Too many arguments")
		return subcommands.ExitFailure
	}

	if err = id.Decode(flag.Arg(1)); err != nil {
		log.Printf("Blob ID parsing error: %v", err)
		return subcommands.ExitFailure
	}

	if flag.NArg() == 3 {
		path := flag.Arg(2)
		if fin, err := os.Open(path); err != nil {
			log.Printf("Failed to open %s: %v", path, err)
			return subcommands.ExitFailure
		} else {
			defer fin.Close()
			var finfo os.FileInfo
			finfo, err = fin.Stat()
			if err == nil {
				err = client.PutN(id, fin, finfo.Size())
			}
		}
	} else {
		err = client.Put(id, os.Stdin)
	}

	if err != nil {
		log.Printf("Put(%v) error: %v", id, err)
		return subcommands.ExitFailure
	} else {
		log.Printf("Put(%v) OK", id)
		return subcommands.ExitSuccess
	}
}
