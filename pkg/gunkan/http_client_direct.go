// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func (self *HttpSimpleClient) Init(url string) error {
	// FIXME(jfsmig): Sanitizes the URL
	self.Endpoint = url
	self.Http = http.Client{}
	return nil
}

func (self *HttpSimpleClient) BuildUrl(path string) string {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.Endpoint)
	b.WriteString(path)
	return b.String()
}

func (self *HttpSimpleClient) srvGet(ctx context.Context, tag string) ([]byte, error) {
	req, err := self.makeRequest(ctx, "GET", self.BuildUrl(tag), nil)
	if err != nil {
		return []byte{}, err
	}

	rep, err := self.Http.Do(req)
	if err != nil {
		return []byte{}, err
	}

	defer rep.Body.Close()
	return ioutil.ReadAll(rep.Body)
}

func (self *HttpSimpleClient) Info(ctx context.Context) ([]byte, error) {
	return self.srvGet(ctx, RouteInfo)
}

func (self *HttpSimpleClient) Health(ctx context.Context) ([]byte, error) {
	return self.srvGet(ctx, RouteHealth)
}

func (self *HttpSimpleClient) Metrics(ctx context.Context) ([]byte, error) {
	return self.srvGet(ctx, RouteMetrics)
}

func (self *HttpSimpleClient) makeRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, path, body)
	if err == nil {
		req.Close = true
		if self.UserAgent != "" {
			req.Header.Set("User-Agent", self.UserAgent)
		} else {
			req.Header.Set("User-Agent", "gunkan-http-go-api/1")
		}
		req.Header.Del("Accept-Encoding")
	}
	return req, err
}
