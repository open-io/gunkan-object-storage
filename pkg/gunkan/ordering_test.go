// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

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
