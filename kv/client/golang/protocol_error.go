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

var (
	errNetwork         = errors.New("Network error")
	errBadRequest      = errors.New("Invalid request")
	errNoSuchKey       = errors.New("No such key")
	errServerError     = errors.New("Server error")
	errMalformedReply  = errors.New("Malformed reply")
	errUnexpectedError = errors.New("Unexpected error code")
	errUnexpectedReply = errors.New("Unexpected reply type")
	errProtocolError   = errors.New("Protocol mismatch")
)

func unpackError(message *Message) error {
	err := unpackMaybeMalformedError(message)
	if err != nil {
		return err
	} else {
		return errMalformedReply
	}
}

func unpackMaybeMalformedError(message *Message) error {
	defer func() { recover() }()

	var unionTable flatbuffers.Table
	if !message.Actual(&unionTable) {
		return errProtocolError
	}

	var errorReply ErrorReply
	errorReply.Init(unionTable.Bytes, unionTable.Pos)
	switch errorReply.Code() {
	case 400:
		return errBadRequest
	case 404:
		return errNoSuchKey
	case 500:
		return errServerError
	default:
		return errUnexpectedError
	}
}
