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
	"math/rand"
)

type simpleBalancer struct {
	catalog Catalog
}

var (
	errNotAvailableDataGate   = errors.New("No data gateway available")
	errNotAvailableIndexGate  = errors.New("No index gateway available")
	errNotAvailableBlobStore  = errors.New("No blob store available")
	errNotAvailableIndexStore = errors.New("No inndex store available")
)

func NewBalancerSimple(catalog Catalog) (Balancer, error) {
	return &simpleBalancer{catalog: catalog}, nil
}

func (self *simpleBalancer) PollIndexGate() (string, error) {
	addrv, err := self.catalog.ListIndexGate()
	if err != nil {
		return "", err
	} else if len(addrv) <= 0 {
		return "", errNotAvailableIndexGate
	} else {
		return addrv[rand.Intn(len(addrv))], nil
	}
}

func (self *simpleBalancer) PollDataGate() (string, error) {
	addrv, err := self.catalog.ListDataGate()
	if err != nil {
		return "", err
	} else if len(addrv) <= 0 {
		return "", errNotAvailableDataGate
	} else {
		return addrv[rand.Intn(len(addrv))], nil
	}
}

func (self *simpleBalancer) PollBlobStore() (string, error) {
	addrv, err := self.catalog.ListBlobStore()
	if err != nil {
		return "", err
	} else if len(addrv) <= 0 {
		return "", errNotAvailableBlobStore
	} else {
		return addrv[rand.Intn(len(addrv))], nil
	}
}
