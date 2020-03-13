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
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"net/http"
	"strconv"
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
		addr, err := ctx.srv.discovery.PollIndexGate()
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		client, err := gunkan.DialIndex(addr, ctx.srv.config.dirConfig)
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		tab, err := client.List(ctx.req.Context(), gunkan.BaseKeyLatest(bucket, marker), max32)
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
