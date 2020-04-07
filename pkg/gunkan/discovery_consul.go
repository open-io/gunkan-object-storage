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
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	consulapi "github.com/armon/consul-api"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func GetConsulEndpoint() (string, error) {
	return "127.0.0.1", nil
}

func NewDiscoveryConsul(ip string) (Discovery, error) {
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

func (self *consulDiscovery) PollIndexGate() (string, error) {
	_, addrv, err := self.resolver.LookupSRV(context.Background(), ConsulSrvIndexGate, ConsulSrvIndexGate, "service.consul")
	if err != nil {
		return "", err
	} else {
		return peekAddr(addrv)
	}
}

func (self *consulDiscovery) PollDataGate() (string, error) {
	_, addrv, err := self.resolver.LookupSRV(context.Background(), ConsulSrvDataGate, ConsulSrvDataGate, "service.consul")
	if err != nil {
		return "", err
	} else {
		return peekAddr(addrv)
	}
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

func arrayHas(needle string, haystack []string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
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

func peekAddr(addrs []*net.SRV) (string, error) {
	// FIXME(jfs): use a weighted random
	rand.Shuffle(len(addrs), func(i, j int) { addrs[i], addrs[j] = addrs[j], addrs[i] })

	addr := addrs[0]
	idxDot := strings.IndexRune(addr.Target, '.')
	if idxDot == 8 {
		str := addr.Target[:8]
		bin, _ := hex.DecodeString(str)
		return fmt.Sprintf("%d.%d.%d.%d:%d", bin[0], bin[1], bin[2], bin[3], addr.Port), nil
	} else if idxDot == 32 {
		return "", errors.New("IPv6 not managed")
	} else {
		return "", errors.New("Adress scheme not managed")
	}
}
