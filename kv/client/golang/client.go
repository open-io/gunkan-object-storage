//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package kv

import (
	_ "nanomsg.org/go/mangos/v2/transport/all"
)

type ListItem struct {
	Key     string
	Version uint64
}

type Client interface {
	Close() error
	Ping() error
	Put(base, key string, value []byte) error
	Get(base, key string) ([]byte, error)
	List(base, marker string) ([]ListItem, error)
}

type Transport interface {
	Close() error
	Send(b []byte) error
	Recv() ([]byte, error)
}

type directClient struct {
	transport Transport
}

func MakeClient(t Transport) (Client, error) {
	//return &logClient{actual: &directClient{transport: t}}, nil
	return &directClient{transport: t}, nil
}

func (self *directClient) Close() error {
	return self.transport.Close()
}

func Dial(url string) (Client, error) {
	t, err := MakeNngSocket(url)
	if err != nil {
		return nil, err
	}
	return MakeClient(t)
}
