// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_data_gate

import (
	"errors"
	"fmt"
	ghttp "github.com/jfsmig/object-storage/internal/helpers-http"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (srv *service) handlePart() ghttp.RequestHandler {
	return func(ctx *ghttp.RequestContext) {
		pre := time.Now()
		id := ctx.Req.URL.Path[len(prefixData):]
		switch ctx.Method() {
		case "GET", "HEAD":
			srv.handleBlobGet(ctx, id)
			srv.timeGet.Observe(time.Since(pre).Seconds())
		case "PUT":
			srv.handleBlobPut(ctx, id)
			srv.timePut.Observe(time.Since(pre).Seconds())
		case "DELETE":
			srv.handleBlobDel(ctx, id)
			srv.timeDel.Observe(time.Since(pre).Seconds())
		default:
			ctx.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (srv *service) handleList() ghttp.RequestHandler {
	h := func(ctx *ghttp.RequestContext) {
		// Unpack request attributes
		q := ctx.Req.URL.Query()
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
		addr, err := srv.lb.PollIndexGate()
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		client, err := gunkan.DialIndexGrpc(addr, srv.config.dirConfig)
		if err != nil {
			ctx.WriteHeader(http.StatusInternalServerError)
			return
		}

		tab, err := client.List(ctx.Req.Context(), gunkan.BK(bucket, marker), max32)
		if err != nil {
			ctx.ReplyError(err)
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
	return func(ctx *ghttp.RequestContext) {
		pre := time.Now()
		h(ctx)
		srv.timeList.Observe(time.Since(pre).Seconds())
	}
}

func (srv *service) handleBlobDel(ctx *ghttp.RequestContext, blobid string) {
	ctx.WriteHeader(http.StatusNotImplemented)
}

func (srv *service) handleBlobGet(ctx *ghttp.RequestContext, blobid string) {
	ctx.WriteHeader(http.StatusNotImplemented)
}

func (srv *service) handleBlobPut(ctx *ghttp.RequestContext, tail string) {
	var err error
	var policy string

	// Unpack the object name
	tokens := strings.Split(tail, "/")
	if len(tokens) != 3 {
		ctx.ReplyCodeError(http.StatusBadRequest, errors.New("3 tokens expected"))
		return
	}

	var id gunkan.BlobId
	id.Bucket = tokens[0]
	id.Content = tokens[1]
	id.PartId = tokens[2]

	// Locate the storage policy
	policy = ctx.Req.Header.Get(HeaderNameObjectPolicy)
	if policy == "" {
		policy = "single"
	}

	// Find a set of backends
	// FIXME(jfsmig): Dumb implementation that only accept the "SINGLE COPY" policy
	var url string
	url, err = srv.lb.PollBlobStore()
	if err != nil {
		ctx.ReplyCodeError(http.StatusServiceUnavailable, err)
		return
	}

	var realid string
	var client gunkan.BlobClient
	client, err = gunkan.DialBlob(url)
	if err != nil {
		ctx.ReplyCodeError(http.StatusInternalServerError, err)
		return
	}

	realid, err = client.Put(ctx.Req.Context(), id, ctx.Input())
	if err != nil {
		ctx.ReplyCodeError(http.StatusServiceUnavailable, err)
		return
	}
	ctx.SetHeader(HeaderPrefixCommon+"part-read-id", realid)
	ctx.WriteHeader(http.StatusCreated)
}
