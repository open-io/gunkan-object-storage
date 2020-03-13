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
	"encoding/json"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type config struct {
	uuid         string
	addrBind     string
	addrAnnounce string
	dirConfig    string
}

type srvStats struct {
	hits uint64 `json:"r"`
	time uint64 `json:"t"`
}

type service struct {
	config    config
	stats     srvStats
	discovery gunkan.Discovery
}

func NewService(cfg config) (*service, error) {
	var err error
	srv := service{config: cfg}
	srv.discovery, err = gunkan.NewDiscovery()
	if err != nil {
		return nil, err
	} else {
		return &srv, nil
	}
}

func get(h handler) handler {
	return func(ctx *handlerContext) {
		switch ctx.req.Method {
		case "GET", "HEAD":
			h(ctx)
		default:
			ctx.rep.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func wrap(srv *service, h handler) http.HandlerFunc {
	return func(rep http.ResponseWriter, req *http.Request) {
		ctx := handlerContext{srv: srv, req: req, rep: rep}
		h(&ctx)
		srv.stats.merge(ctx.stats)
	}
}

func (ctx *handlerContext) Method() string {
	return ctx.req.Method
}

func (ctx *handlerContext) WriteHeader(code int) {
	ctx.rep.WriteHeader(code)
}

func (ctx *handlerContext) Write(b []byte) (int, error) {
	return ctx.rep.Write(b)
}

func (ctx *handlerContext) setHeader(k, v string) {
	ctx.rep.Header().Set(k, v)
}

func (ctx *handlerContext) Input() io.Reader {
	return ctx.req.Body
}

func (ctx *handlerContext) Output() io.Writer {
	return ctx.rep
}

func (ctx *handlerContext) replyCodeErrorMsg(code int, err string) {
	ctx.setHeader("X-Error", err)
	ctx.WriteHeader(code)
}

func (ctx *handlerContext) replyCodeError(code int, err error) {
	ctx.replyCodeErrorMsg(code, err.Error())
}

func (ctx *handlerContext) replyError(err error) {
	code := http.StatusInternalServerError
	if os.IsNotExist(err) {
		code = http.StatusNotFound
	} else if os.IsExist(err) {
		code = http.StatusConflict
	} else if os.IsPermission(err) {
		code = http.StatusForbidden
	} else if os.IsTimeout(err) {
		code = http.StatusRequestTimeout
	}
	ctx.replyCodeErrorMsg(code, err.Error())
}

func (ctx *handlerContext) JSON(o interface{}) {
	ctx.setHeader("Content-Type", "text/plain")
	json.NewEncoder(ctx.Output()).Encode(o)
}

func (ctx *handlerContext) success() {
	ctx.WriteHeader(http.StatusNoContent)
}

func (srv *service) isOverloaded(now time.Time) bool {
	return false
}

func (ss *srvStats) merge(o srvStats) {
	if o.time > 0 {
		atomic.AddUint64(&ss.time, o.time)
	}
	if o.hits > 0 {
		atomic.AddUint64(&ss.hits, o.hits)
	}
}
