// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
