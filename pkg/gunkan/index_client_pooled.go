//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan

import (
	"context"
	"errors"
)

const (
	IndexClientPoolSize = 2
)

type IndexPooledClient struct {
	dirConfig string
	discovery Discovery
	pool      chan IndexClient
	remaining chan bool
}

func DialIndexPooled(dirConfig string) (IndexClient, error) {
	var err error
	var client IndexPooledClient

	client.discovery, err = NewDiscoveryDefault()
	if err != nil {
		return nil, err
	}

	client.dirConfig = dirConfig
	client.pool = make(chan IndexClient, IndexClientPoolSize)
	client.remaining = make(chan bool, IndexClientPoolSize)
	for i := 0; i < IndexClientPoolSize; i++ {
		client.remaining <- true
	}
	close(client.remaining)
	return &client, nil
}

func (self *IndexPooledClient) Get(ctx context.Context, key BaseKey) (string, error) {
	client, err := self.acquire(ctx)
	defer self.release(client)
	if err != nil {
		return "", err
	} else {
		return client.Get(ctx, key)
	}
}

func (self *IndexPooledClient) List(ctx context.Context, key BaseKey, max uint32) ([]string, error) {
	client, err := self.acquire(ctx)
	defer self.release(client)
	if err != nil {
		return nil, err
	} else {
		return client.List(ctx, key, max)
	}
}

func (self *IndexPooledClient) Put(ctx context.Context, key BaseKey, value string) error {
	client, err := self.acquire(ctx)
	defer self.release(client)
	if err != nil {
		return err
	} else {
		return client.Put(ctx, key, value)
	}
}

func (self *IndexPooledClient) Delete(ctx context.Context, key BaseKey) error {
	client, err := self.acquire(ctx)
	defer self.release(client)
	if err != nil {
		return err
	} else {
		return client.Delete(ctx, key)
	}
}

func (self *IndexPooledClient) dial(ctx context.Context) (IndexClient, error) {
	url, err := self.discovery.PollIndexGate()
	if err != nil {
		return nil, err
	}

	return DialIndexGrpc(url, self.dirConfig)
}

func (self *IndexPooledClient) acquire(ctx context.Context) (IndexClient, error) {
	// Item immediately ready from the pool
	select {
	case client := <-self.pool:
		Logger.Debug().Msg("Reusing a direct client")
		return client, nil
	default:
	}

	// Permission to allocate a new item
	ok, _ := <-self.remaining
	if ok {
		Logger.Debug().Msg("Dialing a new direct client")
		return self.dial(ctx)
	}

	// No item, No permission ... wait for an item to be released
	done := ctx.Done()
	Logger.Debug().Msg("Waiting for an idle client")
	select {
	case client := <-self.pool:
		return client, nil
	case <-done:
		return nil, errors.New("No client ready")
	}
}

func (self *IndexPooledClient) release(c IndexClient) {
	if c != nil {
		self.pool <- c
	}
}
