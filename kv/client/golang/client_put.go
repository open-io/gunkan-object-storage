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
	"time"
)

func (self *directClient) Put(base, key string, value []byte) error {
	rawReq := packPutRequest(base, key, value)

	if err := self.transport.Send(rawReq); err != nil {
		return err
	}

	if rawRep, err := self.transport.Recv(); err != nil {
		return err
	} else {
		var rc bool
		rc, err := unpackPutReply(rawRep)
		if !rc && err == nil {
			return errProtocolError
		} else {
			return err
		}
	}
}

func packPutRequest(base, key string, value []byte) []byte {
	now := time.Now()
	builder := flatbuffers.NewBuilder(128 + len(base) + len(key))

	k := builder.CreateString(key)
	b := builder.CreateString(base)
	v := builder.CreateByteString(value)

	PutRequestStart(builder)
	PutRequestAddBase(builder, b)
	PutRequestAddKey(builder, k)
	PutRequestAddValue(builder, v)
	PutRequestAddVersion(builder, uint64(now.UnixNano()))
	req := GetRequestEnd(builder)

	MessageStart(builder)
	MessageAddReqId(builder, 0)
	MessageAddActualType(builder, MessageKindPutRequest)
	MessageAddActual(builder, req)
	msg := MessageEnd(builder)

	builder.Finish(msg)
	return builder.FinishedBytes()
}

func unpackPutReply(rawRep []byte) (bool, error) {
	defer func() { recover() }()

	replyMsg := GetRootAsMessage(rawRep, 0)
	if replyMsg == nil {
		return false, errMalformedReply
	}

	switch replyMsg.ActualType() {
	case MessageKindErrorReply:
		return false, unpackError(replyMsg)
	case MessageKindPutReply:
		var unionTable flatbuffers.Table
		if !replyMsg.Actual(&unionTable) {
			return false, errProtocolError
		}
		return true, nil
	default:
		return false, errUnexpectedReply
	}
}
