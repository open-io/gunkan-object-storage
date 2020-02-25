//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan_blob_model

import (
	"errors"
	"strconv"
	"strings"
)

const (
	stepContent  = iota
	stepPart     = iota
	stepPosition = iota
	stepBucket   = iota
)

type Id struct {
	Bucket   string
	Content  string
	Part     string
	Position uint
}

func (self *Id) Encode() string {
	var b strings.Builder
	self.EncodeIn(&b)
	return b.String()
}

func (self *Id) EncodeIn(b *strings.Builder) {
	b.Grow(256)
	b.WriteString(self.Content)
	b.WriteRune(',')
	b.WriteString(self.Part)
	b.WriteRune(',')
	b.WriteString(strconv.FormatUint(uint64(self.Position), 10))
	b.WriteRune(',')
	b.WriteString(self.Bucket)
}

func (self *Id) Decode(packed string) error {
	var strpos string
	b := strings.Builder{}
	step := stepContent
	for _, c := range packed {
		switch step {
		case stepContent:
			if c == ',' {
				step = stepPart
				self.Content = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepPart:
			if c == ',' {
				step = stepPosition
				self.Part = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepPosition:
			if c == ',' {
				step = stepBucket
				strpos = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepBucket:
			if c == ',' {
				return errors.New("Invalid BLOB Id")
			} else {
				b.WriteRune(c)
			}
		default:
			panic("Invalid State")
		}
	}
	self.Bucket = b.String()
	u64, err := strconv.ParseUint(strpos, 10, 31)
	self.Position = uint(u64)
	return err
}
