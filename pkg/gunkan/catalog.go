// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

type Catalog interface {
	// Returns the list of all the Data Gate services
	ListDataGate() ([]string, error)

	// Returns the list of all the Index Gate services
	ListIndexGate() ([]string, error)

	// Returns the list of all the Blob Store services
	ListBlobStore() ([]string, error)

	// Returns the list of all the Index Store services
	ListIndexStore() ([]string, error)
}

// Returns a discovery client initiated
func NewCatalogDefault() (Catalog, error) {
	if consul, err := GetConsulEndpoint(); err != nil {
		return nil, err
	} else if discovery, err := NewCatalogConsul(consul); err != nil {
		return nil, err
	} else {
		return discovery, nil
	}
}
