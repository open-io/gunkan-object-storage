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
)

type delCmd struct{}

func (*delCmd) Name() string     { return "del" }
func (*delCmd) Synopsis() string { return "Delete a BLOB from a service" }
func (*delCmd) Usage() string {
	return `
del BLOB_ID [BLOB_ID...]

`
}

func (p *delCmd) SetFlags(f *flag.FlagSet) {}

func delOne(strid string) error {
	var err error
	var id blob.Id

	if err = id.Decode(strid); err != nil {
		return err
	} else {
		return client.Delete(id)
	}
}

func (p *delCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if flag.NArg() < 2 {
		log.Println("Missing Blob ID")
		return subcommands.ExitFailure
	} else if flag.NArg() == 2 {
		id := flag.Arg(1)
		if err := delOne(id); err != nil {
			log.Printf("Delete(%v) error: %v", id, err)
			return subcommands.ExitFailure
		} else {
			log.Printf("Delete(%v) OK", id)
			return subcommands.ExitSuccess
		}
	} else {
		strongError := false

		for i := 1; i < flag.NArg(); i++ {
			id := flag.Arg(i)
			if err := delOne(id); err != nil {
				if err != blob.ErrNotFound {
					strongError = true
				}
				log.Printf("%v %v", id, err)
			} else {
				log.Printf("%v OK", id)
			}
		}

		if strongError {
			return subcommands.ExitFailure
		} else {
			return subcommands.ExitSuccess
		}
	}
}
