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
	"github.com/google/flatbuffers/go"
)

func (self *directClient) Ping() error {
	rawReq := packPingRequest()

	if err := self.transport.Send(rawReq); err != nil {
		return err
	}

	if rawRep, err := self.transport.Recv(); err != nil {
		return err
	} else {
		return unpackPingReply(rawRep)
	}
}

func packPingRequest() []byte {
	builder := flatbuffers.NewBuilder(64)

	PingRequestStart(builder)
	req := GetRequestEnd(builder)

	MessageStart(builder)
	MessageAddActualType(builder, MessageKindPingRequest)
	MessageAddActual(builder, req)
	msg := MessageEnd(builder)

	builder.Finish(msg)
	return builder.FinishedBytes()
}

func unpackPingReply(rawRep []byte) error {
	defer func() { recover() }()

	replyMsg := GetRootAsMessage(rawRep, 0)
	if replyMsg == nil {
		return errMalformedReply
	}

	switch replyMsg.ActualType() {
	case MessageKindErrorReply:
		return unpackError(replyMsg)
	case MessageKindPingReply:
		var unionTable flatbuffers.Table
		if !replyMsg.Actual(&unionTable) {
			return errProtocolError
		}
		return nil
	default:
		return errUnexpectedReply
	}
}
