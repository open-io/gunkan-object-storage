//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_data_gate

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type handlerContext struct {
	srv   *service
	req   *http.Request
	rep   http.ResponseWriter
	code  int
	stats srvStats
}

type handler func(ctx *handlerContext)

func handleInfo() handler {
	return func(ctx *handlerContext) {
		ctx.setHeader("Content-Type", "text/plain")
		_, _ = ctx.Write([]byte(infoString))
	}
}

func handleStatus() handler {
	return func(ctx *handlerContext) {
		var st = ctx.srv.stats
		ctx.JSON(&st)
	}
}

func handleHealth() handler {
	return func(ctx *handlerContext) {
		ctx.WriteHeader(http.StatusNoContent)
	}
}

func handleBlob() handler {
	return func(ctx *handlerContext) {
		id := ctx.req.URL.Path[len(prefixBlob):]
		switch ctx.Method() {
		case "GET", "HEAD":
			handleBlobGet(ctx, id)
		case "PUT":
			handleBlobPut(ctx, id)
		case "DELETE":
			handleBlobDel(ctx, id)
		default:
			ctx.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

type dnsDiscovery struct {
	resolver net.Resolver
}

func (self *dnsDiscovery) Init() {
	self.resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		local, e0 := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		remote, e1 := net.ResolveUDPAddr("udp", "127.0.0.1:8600")
		if e0 != nil || e1 != nil {
			return nil, errors.New("Resolution error")
		}
		return net.DialUDP("udp", local, remote)
	}
}

func PollIndex() (string, error) {
	d := dnsDiscovery{}
	d.Init()
	return d.PollIndex()
}

func (self *dnsDiscovery) PollIndex() (string, error) {
	_, addrv, err := self.resolver.LookupSRV(context.Background(), "_"+gunkan.ConsulSrvIndex, "_"+gunkan.ConsulSrvIndex, "service.consul")
	if err != nil {
		return "", err
	} else {
		return peekAddr(addrv)
	}
}

func (self *dnsDiscovery) PollDataProxy() (string, error) {
	_, addrv, err := self.resolver.LookupSRV(context.Background(), "_"+gunkan.ConsulSrvData, "_"+gunkan.ConsulSrvData, "service.consul")
	if err != nil {
		return "", err
	} else {
		return peekAddr(addrv)
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
		return fmt.Sprintf("%d.%d.%d.%d", bin[0], bin[1], bin[2], bin[3]), nil
	} else if idxDot == 32 {
		return "", errors.New("IPv6 not managed")
	} else {
		return "", errors.New("Adress scheme not managed")
	}
}

func handleList() handler {
	return func(ctx *handlerContext) {
		// Unpack request attributes
		q := ctx.req.URL.Query()
		bucket := q.Get("b")
		smax := q.Get("max")
		marker := q.Get("m")

		if !gunkan.ValidateBucketName(bucket) || !gunkan.ValidateContentName(marker) {
			ctx.WriteHeader(http.StatusBadRequest)
			return
		}
		max64, err := strconv.ParseUint(smax, 10, 32)
		if err != nil {
			ctx.WriteHeader(http.StatusBadRequest)
			return
		}
		max32 := uint32(max64)

		// Query the index about a slice of items
		addr, err := PollIndex()
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		client, err := gunkan.DialIndex(addr, ctx.srv.config.configDir)
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		tab, err := client.List(ctx.req.Context(), bucket, marker, max32)
		if err != nil {
			ctx.replyError(err)
			return
		}

		if len(tab) <= 0 {
			ctx.WriteHeader(http.StatusNoContent)
		} else {
			for _, item := range tab {
				fmt.Fprintf(ctx.rep, "%s %d\n", item.Key, item.Version)
			}
		}
	}
}

func handleBlobDel(ctx *handlerContext, blobid string) {
	ctx.WriteHeader(http.StatusNotImplemented)
}

func handleBlobGet(ctx *handlerContext, blobid string) {
	ctx.WriteHeader(http.StatusNotImplemented)
}

func handleBlobPut(ctx *handlerContext, encoded string) {
	ctx.WriteHeader(http.StatusNotImplemented)
}
