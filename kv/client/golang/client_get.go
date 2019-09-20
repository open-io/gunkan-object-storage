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
	"errors"
	"github.com/google/flatbuffers/go"
)

func (self *directClient) Get(base, key string) ([]byte, error) {
	rawReq := packGetRequest(base, key)

	if err := self.transport.Send(rawReq); err != nil {
		return nil, err
	}

	if rawRep, err := self.transport.Recv(); err != nil {
		return nil, err
	} else {
		var rc []byte
		rc, err = unpackGetReply(rawRep)
		if rc == nil && err == nil {
			return rc, errors.New("Malformed reply")
		} else {
			return rc, err
		}
	}
}

func packGetRequest(base, key string) []byte {
	builder := flatbuffers.NewBuilder(128 + len(base) + len(key))

	k := builder.CreateString(key)
	b := builder.CreateString(base)

	GetRequestStart(builder)
	GetRequestAddBase(builder, b)
	GetRequestAddKey(builder, k)
	GetRequestAddVersion(builder, 0)
	req := GetRequestEnd(builder)

	MessageStart(builder)
	MessageAddReqId(builder, 0)
	MessageAddActualType(builder, MessageKindGetRequest)
	MessageAddActual(builder, req)
	msg := MessageEnd(builder)

	builder.Finish(msg)
	return builder.FinishedBytes()
}

func unpackGetReply(rawRep []byte) ([]byte, error) {
	defer func() { recover() }()

	replyMsg := GetRootAsMessage(rawRep, 0)
	if replyMsg == nil {
		return nil, errMalformedReply
	}

	switch replyMsg.ActualType() {
	case MessageKindErrorReply:
		return nil, unpackError(replyMsg)
	case MessageKindGetReply:
		var unionTable flatbuffers.Table
		if !replyMsg.Actual(&unionTable) {
			return nil, errProtocolError
		}
		var r GetReply
		r.Init(unionTable.Bytes, unionTable.Pos)
		return r.Value(), nil
	default:
		return nil, errUnexpectedReply
	}
}
