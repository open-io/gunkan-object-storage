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
	"fmt"
	"strings"
)

type BaseKey struct {
	Key  string
	Base string
}

type ArrayOfKey []string

func (v ArrayOfKey) Len() int      { return len(v) }
func (v ArrayOfKey) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v ArrayOfKey) Less(i, j int) bool {
	return strings.Compare(v[i], v[j]) < 0
}

func BK(base, key string) BaseKey {
	return BaseKey{Base: base, Key: key}
}

func (n *BaseKey) Reset() {
	n.Key = n.Key[:]
	n.Base = n.Base[:]
}

func (n BaseKey) Encode() string {
	return fmt.Sprintf("%s,%s", n.Base, n.Key)
}

func (n *BaseKey) DecodeBytes(b []byte) error {
	return n.DecodeString(string(b))
}

func (n *BaseKey) DecodeString(s string) error {
	step := parsingBase
	sb := strings.Builder{}
	sb.Grow(256)

	n.Reset()
	for _, c := range s {
		switch step {
		case parsingBase:
			if c == ',' {
				n.Base = sb.String()
				sb.Reset()
				step = parsingKey
			} else {
				sb.WriteRune(c)
			}
		case parsingKey:
			sb.WriteRune(c)
		}
	}

	switch step {
	case parsingBase:
		n.Base = sb.String()
	case parsingKey:
		n.Key = sb.String()
	}
	return nil
}

const (
	parsingBase = iota
	parsingKey  = iota
)
