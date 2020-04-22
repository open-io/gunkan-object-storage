//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

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
