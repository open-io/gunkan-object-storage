// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"errors"
	"strconv"
	"strings"
)

const (
	stepBlobParsingContent  = 0
	stepBlobParsingPart     = 1
	stepBlobParsingPosition = 2
	stepBlobParsingBucket   = 3
)

type BlobId struct {
	Bucket   string
	Content  string
	PartId   string
	Position uint
}

func (self BlobId) EncodeMarker() string {
	var b strings.Builder
	self.EncodeMarkerIn(&b)
	return b.String()
}

func (self BlobId) EncodeMarkerIn(b *strings.Builder) {
	b.Grow(256)
	b.WriteString(self.Bucket)
	if len(self.Content) > 0 {
		b.WriteRune(',')
		b.WriteString(self.Content)
		if len(self.PartId) > 0 {
			b.WriteRune(',')
			b.WriteString(self.PartId)
		}
	}
}

func (self BlobId) Encode() string {
	var b strings.Builder
	self.EncodeIn(&b)
	return b.String()
}

func (self BlobId) EncodeIn(b *strings.Builder) {
	b.Grow(256)
	b.WriteString(self.Bucket)
	b.WriteRune(',')
	b.WriteString(self.Content)
	b.WriteRune(',')
	b.WriteString(self.PartId)
	b.WriteRune(',')
	b.WriteString(strconv.FormatUint(uint64(self.Position), 10))
}

func DecodeBlobId(packed string) (BlobId, error) {
	var id BlobId
	b := strings.Builder{}
	step := stepBlobParsingBucket
	for _, c := range packed {
		switch step {
		case stepBlobParsingBucket:
			if c == ',' {
				step = stepBlobParsingContent
				id.Bucket = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepBlobParsingContent:
			if c == ',' {
				step = stepBlobParsingPart
				id.Content = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepBlobParsingPart:
			if c == ',' {
				step = stepBlobParsingPosition
				id.PartId = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepBlobParsingPosition:
			if c == ',' {
				return id, errors.New("Invalid BLOB id")
			} else {
				b.WriteRune(c)
			}
		default:
			panic("Invalid State")
		}
	}

	u64, err := strconv.ParseUint(b.String(), 10, 31)
	id.Position = uint(u64)
	return id, err
}
