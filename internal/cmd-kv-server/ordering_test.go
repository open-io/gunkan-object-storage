//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_server

import (
	"sort"
	"testing"
)

type SetOfVersioned []KeyVersion

func (s SetOfVersioned) Len() int {
	return len(s)
}

func (s SetOfVersioned) Less(i, j int) bool {
	return s[i].Encode() < s[j].Encode()
}

func (s SetOfVersioned) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func TestKeyOrdering(t *testing.T) {
	tab := SetOfVersioned{
		{"A", "plap", 4, true},
		{"A", "plap", 3, true},
		{"A", "plip", 3, true},
		{"A", "plip", 2, false},
		{"A", "plip", 1, true},
		{"A", "plip", 0, true},
		{"A", "plipA", 1, true},
	}
	if !sort.IsSorted(tab) {
		t.Fatal()
	}
}
