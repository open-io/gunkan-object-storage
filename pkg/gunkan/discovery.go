//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan

type Discovery interface {
	// Returns the URL of an available Data Gate service
	PollDataGate() (string, error)

	// Returns the URL of an available Data Gate service
	PollBlobStore() (string, error)

	// Returns the URL of an available Index Gate service
	PollIndexGate() (string, error)

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
func NewDiscoveryDefault() (Discovery, error) {
	if consul, err := GetConsulEndpoint(); err != nil {
		return nil, err
	} else if discovery, err := NewDiscoveryConsul(consul); err != nil {
		return nil, err
	} else {
		return discovery, nil
	}
}
