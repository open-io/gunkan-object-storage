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
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Structure destined to be transported as a query result.
type KeyVersion struct {
	Key     string
	Version uint64
	Active  bool
}

type BaseKeyVersion struct {
	KeyVersion
	Base string
}

func BaseKey(base, key string, version uint64) BaseKeyVersion {
	bkv := BaseKeyVersion{Base: base}
	bkv.Key = key
	bkv.Version = version
	bkv.Active = true
	return bkv
}

func BaseKeyLatest(base, key string) BaseKeyVersion {
	return BaseKey(base, key, math.MaxUint64)
}

func (n BaseKeyVersion) Encode() string {
	v := math.MaxUint64 - n.Version
	if n.Active {
		return fmt.Sprintf("%s,%s,%X", n.Base, n.Key, v)
	} else {
		return fmt.Sprintf("%s,%s,%X#", n.Base, n.Key, v)
	}
}

func (n *BaseKeyVersion) DecodeBytes(b []byte) error {
	return n.DecodeString(string(b))
}

func (n *BaseKeyVersion) DecodeString(s string) error {
	step := parsingBase
	sb := strings.Builder{}
	sb.Grow(256)

	handleVersion := func() error {
		var err error
		n.Version, err = strconv.ParseUint(sb.String(), 10, 63)
		return err
	}

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
			if c == ',' {
				n.Key = sb.String()
				sb.Reset()
				step = parsingVersion
			} else {
				sb.WriteRune(c)
			}
		case parsingVersion:
			if c == ',' {
				if err := handleVersion(); err != nil {
					return err
				}
				sb.Reset()
				step = parsingActive
			} else {
				sb.WriteRune(c)
			}
		case parsingActive:
			if c == '#' {
				n.Active = true
				sb.Reset()
				step = parsingDone
			} else {
				return errors.New("Malformed Key (unexpected)")
			}
		case parsingDone:
			return errors.New("Malformed Key (trail)")
		}
	}

	switch step {
	case parsingVersion:
		n.Active = true
		return handleVersion()
	case parsingDone:
		return nil
	default:
		return errors.New("Malformed Key")
	}
}

func Key(key string, version uint64) KeyVersion {
	return KeyVersion{Key: key, Version: version, Active: true}
}

func KeyLatest(base, key string) KeyVersion {
	return Key(key, math.MaxUint64)
}

func (n KeyVersion) Encode() string {
	v := math.MaxUint64 - n.Version
	if n.Active {
		return fmt.Sprintf("%s,%X", n.Key, v)
	} else {
		return fmt.Sprintf("%s,%X#", n.Key, v)
	}
}

func (n *KeyVersion) DecodeBytes(b []byte) error {
	return n.DecodeString(string(b))
}

func (n *KeyVersion) DecodeString(s string) error {
	step := parsingKey
	sb := strings.Builder{}
	sb.Grow(256)

	handleVersion := func() error {
		var err error
		n.Version, err = strconv.ParseUint(sb.String(), 10, 63)
		return err
	}

	for _, c := range s {
		switch step {
		case parsingKey:
			if c == ',' {
				n.Key = sb.String()
				sb.Reset()
				step = parsingVersion
			} else {
				sb.WriteRune(c)
			}
		case parsingVersion:
			if c == ',' {
				if err := handleVersion(); err != nil {
					return err
				}
				sb.Reset()
				step = parsingActive
			} else {
				sb.WriteRune(c)
			}
		case parsingActive:
			if c == '#' {
				n.Active = true
				sb.Reset()
				step = parsingDone
			} else {
				return errors.New("Malformed Key (unexpected)")
			}
		case parsingDone:
			return errors.New("Malformed Key (trail)")
		}
	}

	switch step {
	case parsingVersion:
		n.Active = true
		return handleVersion()
	case parsingDone:
		return nil
	default:
		return errors.New("Malformed Key")
	}
}

const (
	parsingBase    = iota
	parsingKey     = iota
	parsingVersion = iota
	parsingActive  = iota
	parsingDone    = iota
)
