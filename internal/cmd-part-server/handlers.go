//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_part_server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"math/rand"
	"net"
	"net/http"
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
		ctx.WriteHeader(http.StatusNotImplemented)
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

type Discovery struct {
	resolver net.Resolver
}

func (self *Discovery) Init() {
	self.resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		local, e0 := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		remote, e1 := net.ResolveUDPAddr("udp", "127.0.0.1:8600")
		if e0 != nil || e1 != nil {
			return nil, errors.New("Resolution error")
		}
		return net.DialUDP("udp", local, remote)
	}
}

func (self *Discovery) PollIndex() (string, error) {
	_, addrv, err := self.resolver.LookupSRV(context.Background(), "_"+gunkan.ConsulSrvIndex, "_"+gunkan.ConsulSrvIndex, "service.consul")
	if err != nil {
		return "", err
	} else {
		return peekAddr(addrv)
	}
}

func (self *Discovery) PollDataProxy() (string, error) {
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
		// Query the index about a slice of items

		ctx.WriteHeader(http.StatusNotImplemented)
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
