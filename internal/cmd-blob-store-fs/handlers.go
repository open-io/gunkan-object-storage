//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_store_fs

import (
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"golang.org/x/sys/unix"
	"io"
	"net/http"
	"time"
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
		now := time.Now()
		if ctx.srv.isError(now) {
			ctx.replyCodeErrorMsg(http.StatusBadGateway, "Recent I/O errors")
		} else if ctx.srv.isFull(now) {
			// Consul reacts to http.StatusTooManyRequests with a warning but
			// no error. It seems a better alternative to http.StatusInsufficientStorage
			// that would mark the service as failed.
			ctx.replyCodeErrorMsg(http.StatusTooManyRequests, "Full")
		} else if ctx.srv.isOverloaded(now) {
			ctx.replyCodeErrorMsg(http.StatusTooManyRequests, "Overloaded")
		} else {
			ctx.WriteHeader(http.StatusNoContent)
		}
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
		switch ctx.Method() {
		case "GET", "HEAD":
			ctx.WriteHeader(http.StatusNotImplemented)
		default:
			ctx.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleBlobDel(ctx *handlerContext, blobid string) {
	err := ctx.srv.repo.Delete(blobid)
	if err != nil {
		ctx.replyError(err)
	} else {
		ctx.success()
	}
}

func handleBlobGet(ctx *handlerContext, blobid string) {
	var st unix.Stat_t
	var f BlobReader
	var err error

	f, err = ctx.srv.repo.Open(blobid)
	if err != nil {
		ctx.replyError(err)
		return
	} else {
		defer f.Close()
	}

	err = unix.Fstat(int(f.Stream().Fd()), &st)
	if err == nil {
		if st.Size == 0 {
			ctx.setHeader("Content-Length", "0")
			ctx.WriteHeader(http.StatusNoContent)
		} else {
			ctx.setHeader("Content-Length", fmt.Sprintf("%d", st.Size))
			ctx.WriteHeader(http.StatusOK)
		}
		ctx.setHeader("Content-Type", "octet/stream")
		_, err = io.Copy(ctx.Output(), &io.LimitedReader{R: f.Stream(), N: st.Size})
	}
	if err != nil {
		ctx.replyError(err)
	}
}

func handleBlobPut(ctx *handlerContext, encoded string) {
	objid := gunkan.BlobId{}
	if err := objid.Decode(string(encoded)); err != nil {
		ctx.replyCodeError(http.StatusBadRequest, err)
		return
	}

	f, err := ctx.srv.repo.Create(objid)
	if err != nil {
		ctx.replyError(err)
		return
	}

	var final string
	_, err = io.Copy(f.Stream(), ctx.Input())
	if err != nil {
		f.Abort()
		ctx.replyError(err)
	} else if final, err = f.Commit(); err != nil {
		ctx.replyError(err)
	} else {
		ctx.setHeader("Location", final)
		ctx.WriteHeader(http.StatusCreated)
	}
}
