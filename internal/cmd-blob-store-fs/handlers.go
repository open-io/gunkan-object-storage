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
	ghttp "github.com/jfsmig/object-storage/internal/helpers-http"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"golang.org/x/sys/unix"
	"io"
	"net/http"
	"time"
)

func (srv *service) handleBlob() ghttp.RequestHandler {
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
		ctx.WriteHeader(http.StatusNotImplemented)
	}
	return func(ctx *ghttp.RequestContext) {
		pre := time.Now()
		h(ctx)
		srv.timeList.Observe(time.Since(pre).Seconds())
	}
}

func (srv *service) handleBlobDel(ctx *ghttp.RequestContext, blobid string) {
	err := srv.repo.Delete(blobid)
	if err != nil {
		ctx.ReplyError(err)
	} else {
		ctx.ReplySuccess()
	}
}

func (srv *service) handleBlobGet(ctx *ghttp.RequestContext, blobid string) {
	var st unix.Stat_t
	var f BlobReader
	var err error

	f, err = srv.repo.Open(blobid)
	if err != nil {
		ctx.ReplyError(err)
		return
	} else {
		defer f.Close()
	}

	err = unix.Fstat(int(f.Stream().Fd()), &st)
	if err == nil {
		if st.Size == 0 {
			ctx.SetHeader("Content-Length", "0")
			ctx.WriteHeader(http.StatusNoContent)
		} else {
			ctx.SetHeader("Content-Length", fmt.Sprintf("%d", st.Size))
			ctx.WriteHeader(http.StatusOK)
		}
		ctx.SetHeader("Content-Type", "octet/stream")
		_, err = io.Copy(ctx.Output(), &io.LimitedReader{R: f.Stream(), N: st.Size})
	}
	if err != nil {
		ctx.ReplyError(err)
	}
}

func (srv *service) handleBlobPut(ctx *ghttp.RequestContext, encoded string) {
	var err error
	var id gunkan.BlobId

	if id, err = gunkan.DecodeBlobId(string(encoded)); err != nil {
		ctx.ReplyCodeError(http.StatusBadRequest, err)
		return
	}

	f, err := srv.repo.Create(id)
	if err != nil {
		ctx.ReplyError(err)
		return
	}

	var final string
	_, err = io.Copy(f.Stream(), ctx.Input())
	if err != nil {
		f.Abort()
		ctx.ReplyError(err)
	} else if final, err = f.Commit(); err != nil {
		ctx.ReplyError(err)
	} else {
		ctx.SetHeader("Location", final)
		ctx.WriteHeader(http.StatusCreated)
	}
}
