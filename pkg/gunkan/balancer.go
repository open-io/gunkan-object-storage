//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan

type Balancer interface {
	// Returns the URL of an available Data Gate service
	PollDataGate() (string, error)

	// Returns the URL of an available Index Gate service
	PollIndexGate() (string, error)

	// Returns the URL of an available Blob Store service.
	PollBlobStore() (string, error)
}

// Returns a discovery client initiated
func NewBalancerDefault() (Balancer, error) {
	if catalog, err := NewCatalogDefault(); err != nil {
		return nil, err
	} else if discovery, err := NewBalancerSimple(catalog); err != nil {
		return nil, err
	} else {
		return discovery, nil
	}
}
