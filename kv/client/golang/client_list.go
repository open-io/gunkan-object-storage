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

func (self *directClient) List(base, marker string) ([]ListItem, error) {
	rawReq := packListRequest(base, marker)

	if err := self.transport.Send(rawReq); err != nil {
		return nil, err
	}

	if rawRep, err := self.transport.Recv(); err != nil {
		return nil, err
	} else {
		var rc []ListItem
		rc, err = unpackListReply(rawRep)
		if rc == nil && err == nil {
			return rc, errProtocolError
		} else {
			return rc, err
		}
	}
}

func packListRequest(base, marker string) []byte {
	builder := flatbuffers.NewBuilder(128 + len(base) + len(marker))

	b := builder.CreateString(base)
	k := builder.CreateString(marker)

	ListRequestStart(builder)
	ListRequestAddBase(builder, b)
	ListRequestAddMarker(builder, k)
	ListRequestAddMax(builder, 0)
	req := ListRequestEnd(builder)

	MessageStart(builder)
	MessageAddReqId(builder, 0)
	MessageAddActualType(builder, MessageKindListRequest)
	MessageAddActual(builder, req)
	msg := MessageEnd(builder)

	builder.Finish(msg)
	return builder.FinishedBytes()
}

func unpackListReply(rawRep []byte) ([]ListItem, error) {
	defer func() { recover() }()

	replyMsg := GetRootAsMessage(rawRep, 0)
	if replyMsg == nil {
		return nil, errMalformedReply
	}
	switch replyMsg.ActualType() {
	case MessageKindErrorReply:
		return nil, unpackError(replyMsg)
	case MessageKindListReply:
		var unionTable flatbuffers.Table
		if !replyMsg.Actual(&unionTable) {
			return nil, errProtocolError
		}
		var r ListReply
		r.Init(unionTable.Bytes, unionTable.Pos)
		max := r.ItemsLength()
		result := make([]ListItem, 0)
		for i := 0; i < max; i++ {
			var le ListEntry
			r.Items(&le, i)
			result = append(result, ListItem{Key: string(le.Key()), Version: le.Version()})
		}
		return result, nil
	default:
		return nil, errUnexpectedReply
	}
}
