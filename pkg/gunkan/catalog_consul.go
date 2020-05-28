// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
	"errors"
	consulapi "github.com/armon/consul-api"
	"net"
	"strconv"
)

func GetConsulEndpoint() (string, error) {
	return "127.0.0.1", nil
}

func NewCatalogConsul(ip string) (Catalog, error) {
	d := consulDiscovery{}
	if err := d.init(ip); err != nil {
		return nil, err
	} else {
		return &d, nil
	}
}

type consulDiscovery struct {
	resolver net.Resolver
	consul   *consulapi.Client
}

func (self *consulDiscovery) init(ip string) error {
	self.resolver.PreferGo = false
	self.resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		local, e0 := net.ResolveUDPAddr("udp", ip+":0")
		remote, e1 := net.ResolveUDPAddr("udp", ip+":8600")
		if e0 != nil || e1 != nil {
			return nil, errors.New("Resolution error")
		}
		return net.DialUDP("udp", local, remote)
	}

	cfg := consulapi.Config{}
	if endpoint, err := GetConsulEndpoint(); err != nil {
		return err
	} else {
		cfg.Address = endpoint + ":8500"
	}
	var err error
	self.consul, err = consulapi.NewClient(&cfg)
	return err
}

func (self *consulDiscovery) ListIndexGate() ([]string, error) {
	return self.listServices(ConsulSrvIndexGate)
}

func (self *consulDiscovery) ListDataGate() ([]string, error) {
	return self.listServices(ConsulSrvDataGate)
}

func (self *consulDiscovery) ListIndexStore() ([]string, error) {
	return self.listServices(ConsulSrvIndexStore)
}

func (self *consulDiscovery) ListBlobStore() ([]string, error) {
	return self.listServices(ConsulSrvBlobStore)
}

func (self *consulDiscovery) listServices(srvtype string) ([]string, error) {
	var result []string
	args := consulapi.QueryOptions{}
	args.Datacenter = ""
	args.AllowStale = true
	args.RequireConsistent = false
	catalog := self.consul.Catalog()
	if srvtab, _, err := catalog.Services(&args); err != nil {
		return result, err
	} else {
		for srvid, tags := range srvtab {
			if !arrayHas(srvtype, tags[:]) {
				continue
			}
			allsrv, _, err := catalog.Service(srvid, srvtype, &args)
			if err != nil {
				Logger.Info().Str("id", srvid).Str("type", srvtype).Err(err).Msg("Service resolution error")
			} else {
				for _, srv := range allsrv {
					result = append(result, srv.Address+":"+strconv.Itoa(srv.ServicePort))
				}
			}
		}
		Logger.Debug().Str("type", srvtype).Int("nb", len(result)).Msg("Loaded")
		return result, nil
	}
}

func arrayHas(needle string, haystack []string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}
