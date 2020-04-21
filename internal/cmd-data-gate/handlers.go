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
	"errors"
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
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
		gunkan.Logger.Debug().Str("method", ctx.req.Method).Msg("blob request")
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

		client, err := gunkan.DialIndexGrpc(addr, ctx.srv.config.dirConfig)
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		tab, err := client.List(ctx.req.Context(), gunkan.BK(bucket, marker), max32)
		if err != nil {
			ctx.replyError(err)
			return
		}

		if len(tab) <= 0 {
			ctx.WriteHeader(http.StatusNoContent)
		} else {
			for _, item := range tab {
				fmt.Println(item)
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

const (
	HeaderPrefixCommon     = "X-gk-"
	HeaderNameObjectPolicy = HeaderPrefixCommon + "obj-policy"
)

func handleBlobPut(ctx *handlerContext, tail string) {
	var err error
	var policy string

	gunkan.Logger.Warn().Str("url", tail).Msg("PUT")

	// Unpack the object name
	tokens := strings.Split(tail, "/")
	if len(tokens) != 3 {
		ctx.replyCodeError(http.StatusBadRequest, errors.New("3 tokens expected"))
		return
	}

	var id gunkan.BlobId
	id.Bucket = tokens[0]
	id.Content = tokens[1]
	id.PartId = tokens[2]

	// Locate the storage policy
	policy = ctx.req.Header.Get(HeaderNameObjectPolicy)
	if policy == "" {
		policy = "single"
	}

	gunkan.Logger.Warn().Str("pol", policy).Str("obj", id.Encode()).Msg("PUT")

	// Find a set of backends
	// FIXME(jfsmig): Dumb implementation that only accept the "SINGLE COPY" policy
	var url string
	url, err = ctx.srv.discovery.PollBlobStore()
	if err != nil {
		ctx.replyCodeError(http.StatusServiceUnavailable, err)
		return
	}

	var realid string
	var client gunkan.BlobClient
	client, err = gunkan.DialBlob(url)
	if err != nil {
		ctx.replyCodeError(http.StatusInternalServerError, err)
		return
	}
	defer client.Close()

	realid, err = client.Put(ctx.req.Context(), id, ctx.Input())
	if err != nil {
		ctx.replyCodeError(http.StatusServiceUnavailable, err)
		return
	}
	ctx.rep.Header().Set(HeaderPrefixCommon+"part-read-id", realid)
	ctx.WriteHeader(http.StatusNotImplemented)
}
