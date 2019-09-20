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
	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/req"
	_ "nanomsg.org/go/mangos/v2/transport/all"
	"time"
)

type NngSocket struct {
	socket mangos.Socket
}

func (self *NngSocket) Close() error {
	return self.socket.Close()
}

func (self *NngSocket) Recv() ([]byte, error) {
	deadline := time.Now().Add(time.Second)
	self.socket.SetOption(mangos.OptionRecvDeadline, deadline)
	return self.socket.Recv()
}

func (self *NngSocket) Send(buffer []byte) error {
	deadline := time.Now().Add(time.Second)
	self.socket.SetOption(mangos.OptionSendDeadline, deadline)
	return self.socket.Send(buffer)
}

func MakeNngSocket(url string) (Transport, error) {
	if s, err := req.NewSocket(); err != nil {
		return nil, err
	} else {
		//s.SetOption(mangos.OptionKeepAlive, 1)
		s.SetOption(mangos.OptionNoDelay, 0)
		s.SetOption(mangos.OptionRetryTime, 0)
		s.SetOption(mangos.OptionLinger, time.Second)
		s.SetOption(mangos.OptionMaxRecvSize, 4096 * 1024)
		s.SetOption(mangos.OptionReadQLen, 4096)
		if err := s.Dial(url); err != nil {
			return nil, err
		} else {
			return &NngSocket{socket: s}, nil
		}
	}
}

type NngContext struct {
	context mangos.Context
}

func (self *NngContext) Close() error {
	return self.context.Close()
}

func (self *NngContext) Recv() ([]byte, error) {
	deadline := time.Now().Add(time.Second)
	self.context.SetOption(mangos.OptionRecvDeadline, deadline)
	return self.context.Recv()
}

func (self *NngContext) Send(buffer []byte) error {
	deadline := time.Now().Add(time.Second)
	self.context.SetOption(mangos.OptionSendDeadline, deadline)
	return self.context.Send(buffer)
}

func ShareNngSocket(t Transport) (Transport, error) {
	nngSock := t.(*NngSocket)
	ctx, err := nngSock.socket.OpenContext()
	if err != nil {
		return nil, err
	} else {
		return &NngContext{context: ctx}, nil
	}
}
