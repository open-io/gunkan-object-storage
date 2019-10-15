//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package blob

import(
	"io"
	"errors"
	"strings"
	"strconv"
)

var (
	ErrNotFound = errors.New("404/Not-Found")
	ErrForbidden = errors.New("403/Forbidden")
	ErrAlreadyExists = errors.New("409/Conflict")
	ErrStorageError = errors.New("502/Backend-Error")
	ErrInternalError = errors.New("500/Internal Error")
)

type Id struct {
	Content string
	Part string
	Position uint
}

type Stats struct {
	B_in      uint64 `json:"b_in"`
	B_out     uint64 `json:"b_out"`
	T_info    uint64 `json:"t_info"`
	T_status  uint64 `json:"t_status"`
	T_put     uint64 `json:"t_put"`
	T_get     uint64 `json:"t_get"`
	T_head    uint64 `json:"t_head"`
	T_delete  uint64 `json:"t_delete"`
	T_list    uint64 `json:"t_list"`
	T_other   uint64 `json:"t_other"`
	H_info    uint64 `json:"h_info"`
	H_status  uint64 `json:"h_status"`
	H_put     uint64 `json:"h_put"`
	H_get     uint64 `json:"h_get"`
	H_head    uint64 `json:"h_head"`
	H_delete  uint64 `json:"h_delete"`
	H_list    uint64 `json:"h_list"`
	H_other   uint64 `json:"h_other"`
	C_200     uint64 `json:"c_200"`
	C_201     uint64 `json:"c_201"`
	C_204     uint64 `json:"c_204"`
	C_206     uint64 `json:"c_206"`
	C_400     uint64 `json:"c_400"`
	C_403     uint64 `json:"c_403"`
	C_404     uint64 `json:"c_404"`
	C_405     uint64 `json:"c_405"`
	C_408     uint64 `json:"c_408"`
	C_409     uint64 `json:"c_409"`
	C_418     uint64 `json:"c_418"`
	C_499     uint64 `json:"c_499"`
	C_502     uint64 `json:"c_502"`
	C_503     uint64 `json:"c_503"`
	C_50X     uint64 `json:"c_50X"`
}

type Client interface {
	Close() error
	Status() (Stats, error)
	Put(id Id, data io.Reader) error
	Get(id Id) (io.ReadCloser, error)
	Delete(id Id) error
	List(max uint) ([]Id, error)
	ListAfter(max uint, marker Id) ([]Id, error)
}

func Dial(url string) (Client, error) {
	return newHttpClient(url), nil
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
}

const (
	stepContent = iota
	stepPart = iota
	stepPosition = iota
)

func (self *Id) Decode(packed string) error {
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
				return errors.New("Invalid BLOB Id")
			} else {
				b.WriteRune(c);
			}
		default:
			panic("Invalid State");
		}
	}
	u64, err := strconv.ParseUint(b.String(), 10, 31)
	self.Position = uint(u64)
	return err
}
