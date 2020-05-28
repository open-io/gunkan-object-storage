// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghttp

import (
	"encoding/json"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
	"os"
)

type Service struct {
	Url  string
	Info string

	mux *http.ServeMux
}

type RequestContext struct {
	Srv  *Service
	Req  *http.Request
	Rep  http.ResponseWriter
	Err  error
	Code int
}

type RequestHandler func(ctx *RequestContext)

func NewHttpApi(url, info string) *Service {
	var srv Service
	srv.Url = url
	srv.Info = info

	srv.mux = http.NewServeMux()
	srv.mux.HandleFunc(gunkan.RouteInfo, getF(srv.handleInfo()))
	srv.mux.HandleFunc(gunkan.RouteHealth, getF(srv.handleHealth()))
	srv.mux.Handle(gunkan.RouteMetrics, promhttp.Handler())
	return &srv
}

func (srv *Service) Handler() http.Handler {
	return srv.mux
}

func (srv *Service) Route(pattern string, h RequestHandler) {
	srv.mux.HandleFunc(pattern, srv.wrap(h))
}

func (srv *Service) handleInfo() http.HandlerFunc {
	return func(rep http.ResponseWriter, req *http.Request) {
		rep.Header().Set("Content-Type", "text/plain")
		_, _ = rep.Write([]byte(srv.Info))
	}
}

func (srv *Service) handleHealth() http.HandlerFunc {
	return func(rep http.ResponseWriter, req *http.Request) {
		rep.WriteHeader(http.StatusNoContent)
	}
}

func (srv *Service) wrap(h RequestHandler) http.HandlerFunc {
	return func(rep http.ResponseWriter, req *http.Request) {
		ctx := RequestContext{Req: req, Rep: rep}
		h(&ctx)
		gunkan.Logger.Info().
			Str("local", srv.Url).
			Str("peer", req.RemoteAddr).
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Err(ctx.Err).
			Int("rc", ctx.Code).Msg("access")
	}
}

func (ctx *RequestContext) WriteHeader(code int) {
	ctx.Code = code
	ctx.Rep.WriteHeader(code)
}

func (ctx *RequestContext) Write(b []byte) (int, error) {
	return ctx.Rep.Write(b)
}

func (ctx *RequestContext) SetHeader(k, v string) {
	ctx.Rep.Header().Set(k, v)
}

func (ctx *RequestContext) Method() string {
	return ctx.Req.Method
}

func (ctx *RequestContext) Input() io.Reader {
	return ctx.Req.Body
}

func (ctx *RequestContext) Output() io.Writer {
	return ctx.Rep
}

func (ctx *RequestContext) ReplyCodeErrorMsg(code int, err string) {
	ctx.Code = code
	replySetErrorMsg(ctx.Rep, code, err)
}

func (ctx *RequestContext) ReplyCodeError(code int, err error) {
	ctx.ReplyCodeErrorMsg(code, err.Error())
}

func (ctx *RequestContext) ReplyError(err error) {
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
	ctx.ReplyCodeErrorMsg(code, err.Error())
}

func (ctx *RequestContext) JSON(o interface{}) {
	ctx.SetHeader("Content-Type", "text/plain")
	json.NewEncoder(ctx.Output()).Encode(o)
}

func (ctx *RequestContext) ReplySuccess() {
	ctx.WriteHeader(http.StatusNoContent)
}

func replySetErrorMsg(rep http.ResponseWriter, code int, err string) {
	rep.Header().Set("X-Error", err)
	rep.WriteHeader(code)
}

func getF(h http.HandlerFunc) http.HandlerFunc {
	return func(rep http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET", "HEAD":
			h(rep, req)
		default:
			replySetErrorMsg(rep, http.StatusMethodNotAllowed, "Only GET or HEAD")
		}
	}
}

func Get(h RequestHandler) RequestHandler {
	return func(ctx *RequestContext) {
		switch ctx.Method() {
		case "GET", "HEAD":
			h(ctx)
		default:
			ctx.ReplyCodeErrorMsg(http.StatusMethodNotAllowed, "Only GET or HEAD")
		}
	}
}
