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
	"encoding/json"
	"flag"
	"github.com/google/subcommands"
	"log"
	"os"
)

type statusCmd struct{}

func (*statusCmd) Name() string     { return "status" }
func (*statusCmd) Synopsis() string { return "Collect the usage stats from the service." }
func (*statusCmd) Usage() string {
	return `
status

`
}

func (p *statusCmd) SetFlags(f *flag.FlagSet) {}

func (p *statusCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if st, err := client.Status(); err != nil {
		log.Printf("Status error: %v", err)
		return subcommands.ExitFailure
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(&st)
		return subcommands.ExitSuccess
	}
}

