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
	"math"
	"os"
	"sync"
	"strconv"
	"time"
)

type floodCmd struct{}

func (*floodCmd) Name() string     { return "flood" }
func (*floodCmd) Synopsis() string { return "Flood a KV service with PING commands" }
func (*floodCmd) Usage() string    { return `flood` }

func (p *floodCmd) SetFlags(f *flag.FlagSet) {}

func (p *floodCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(targets) <= 0 {
		log.Printf("No target specified")
		return subcommands.ExitUsageError
	}

	// Slices of 100us
	distribution := make([]int64, 1000, 1000)
	spent := make(chan time.Duration, 32)
	consumeDurations := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for d := range spent {
			distribution[d.Nanoseconds() / 100000] ++
		}
	}

	floodSingle := func(t kv.Transport, wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		client, err := kv.MakeClient(t)
		if err != nil {
			log.Fatalf("Context creation error: %v", err)
		}
		for i := 0; i < 10000; i++ {
			pre := time.Now()
			err = client.Ping()
			spent <- time.Now().Sub(pre)
			if err != nil {
				log.Printf("PING error: %v", err)
				panic("REQ error")
			}
		}
	}
	floodMany := func(wg *sync.WaitGroup) {
		defer wg.Done()
		client, err := kv.MakeNngSocket(targets)
		if err != nil {
			log.Printf("Client connection error: %v", err)
			panic("CNX error")
		}
		defer client.Close()

		var subWg sync.WaitGroup
		for i := 0; i < 10; i++ {
			subWg.Add(1)
			ctx, err := kv.ShareNngSocket(client)
			if err != nil {
				log.Fatalf("Socket creation error: %v", err)
			}
			go floodSingle(ctx, &subWg)
		}
		subWg.Wait()
	}

	var wgStats, wgWorkers sync.WaitGroup
	wgStats.Add(1)
	go consumeDurations(&wgStats)
	for i := 0; i < 10; i++ {
		wgWorkers.Add(1)
		go floodMany(&wgWorkers)
	}
	wgWorkers.Wait()

	close(spent)
	wgStats.Wait()

	max := len(distribution)
	for ; max > 0; max-- {
		if distribution[max-1] > 0 {
			break
		}
	}

	strCols := os.Getenv("COLUMNS")
	iCols, err := strconv.Atoi(strCols)
	if err != nil || iCols <= 0 {
		iCols = 250
	}

	vMax := int64(0)
	for i:=0; i < max; i++ {
		if distribution[i] > vMax {
			vMax = distribution[i]
		}
	}

	for i:=0; i < max; i++ {
		var d time.Duration = time.Duration((i + 1) * 100000)
		v := distribution[i]
/*
		realV := 0.0
		if v > 0 {
			realV = math.Log10(v)
		}
*/
		realV := int(math.Ceil((float64(v) * float64(iCols - 10)) / float64(vMax)))
		bar := ""
		for iDot:=0; iDot < realV ;iDot++ {
			bar = bar + "-"
		}
		fmt.Printf("%0.4f %s\n", d.Seconds(), bar)
	}

	return subcommands.ExitSuccess
}
