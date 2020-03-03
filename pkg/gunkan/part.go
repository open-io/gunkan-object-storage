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
	"errors"
	"strings"
)

const (
	stepPartParsingContent = 0
	stepPartParsingPart    = 1
	stepPartParsingBucket  = 2
)

type PartId struct {
	Bucket  string
	Content string
	PartId  string
}

func (self *PartId) Encode() string {
	var b strings.Builder
	self.EncodeIn(&b)
	return b.String()
}

func (self *PartId) EncodeIn(b *strings.Builder) {
	b.Grow(256)
	b.WriteString(self.Content)
	b.WriteRune(',')
	b.WriteString(self.PartId)
	b.WriteRune(',')
	b.WriteString(self.Bucket)
}

func (self *PartId) EncodeMarker() string {
	var b strings.Builder
	self.EncodeMarkerIn(&b)
	return b.String()
}

func (self *PartId) EncodeMarkerIn(b *strings.Builder) {
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

func (self *PartId) Decode(packed string) error {
	b := strings.Builder{}
	step := stepPartParsingBucket
	for _, c := range packed {
		switch step {
		case stepPartParsingBucket:
			if c == ',' {
				step = stepPartParsingContent
				self.Content = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepPartParsingContent:
			if c == ',' {
				step = stepPartParsingPart
				self.Content = b.String()
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		case stepPartParsingPart:
			if c == ',' {
				return errors.New("Invalid PART id")
			} else {
				b.WriteRune(c)
			}
		default:
			panic("Invalid State")
		}
	}
	self.PartId = b.String()
	return nil
}
