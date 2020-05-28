//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan

import (
	"bufio"
	"context"
	"io"
	"strings"
)

type httpPartClient struct {
	client HttpSimpleClient
}

func DialPart(url string) (PartClient, error) {
	var err error
	var rc httpPartClient
	err = rc.client.Init(url)
	if err != nil {
		return nil, err
	}
	return &rc, nil
}

func (self *httpPartClient) Delete(ctx context.Context, id PartId) error {
	url := self.client.BuildUrl("/v1/blob")

	req, err := self.client.makeRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	rep, err := self.client.Http.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return MapCodeToError(rep.StatusCode)
}

func (self *httpPartClient) Get(ctx context.Context, id PartId) (io.ReadCloser, error) {
	url := self.client.BuildUrl("/v1/blob")

	req, err := self.client.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Http.Do(req)
	if err != nil {
		return nil, err
	}

	switch rep.StatusCode {
	case 200, 201, 204:
		return rep.Body, nil
	default:
		return nil, MapCodeToError(rep.StatusCode)
	}
}

func (self *httpPartClient) PutN(ctx context.Context, id PartId, data io.Reader, size int64) error {
	url := self.client.BuildUrl("/v1/blob")

	req, err := self.client.makeRequest(ctx, "PUT", url, data)
	if err != nil {
		return err
	}

	req.ContentLength = size
	rep, err := self.client.Http.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return MapCodeToError(rep.StatusCode)
}

func (self *httpPartClient) Put(ctx context.Context, id PartId, data io.Reader) error {
	url := self.client.BuildUrl("/v1/blob")

	req, err := self.client.makeRequest(ctx, "PUT", url, data)
	if err != nil {
		return err
	}

	req.ContentLength = -1
	rep, err := self.client.Http.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return MapCodeToError(rep.StatusCode)
}

func (self *httpPartClient) List(ctx context.Context, max uint32) ([]PartId, error) {
	return self.listRaw(ctx, max, "")
}

func (self *httpPartClient) ListAfter(ctx context.Context, max uint32, id PartId) ([]PartId, error) {
	return self.listRaw(ctx, max, id.EncodeMarker())
}

func (self *httpPartClient) listRaw(ctx context.Context, max uint32, marker string) ([]PartId, error) {
	url := self.client.BuildUrl("/v1/list")

	req, err := self.client.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Http.Do(req)
	if err != nil {
		return nil, err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 200, 201, 204:
		return unpackPartIdArray(rep.Body)
	default:
		return nil, MapCodeToError(rep.StatusCode)
	}
}

func unpackPartIdArray(body io.Reader) ([]PartId, error) {
	rc := make([]PartId, 0)
	r := bufio.NewReader(body)
	for {
		if line, err := r.ReadString('\n'); err != nil {
			if err == io.EOF {
				return rc, nil
			} else {
				return nil, err
			}
		} else if len(line) > 0 {
			var id PartId
			line = strings.Trim(line, "\r\n")
			if err = id.Decode(line); err != nil {
				return nil, err
			} else {
				rc = append(rc, id)
			}
		}
	}
}

func (self *httpPartClient) Info(ctx context.Context) ([]byte, error) {
	return self.client.Info(ctx)
}

func (self *httpPartClient) Health(ctx context.Context) ([]byte, error) {
	return self.client.Health(ctx)
}

func (self *httpPartClient) Metrics(ctx context.Context) ([]byte, error) {
	return self.client.Metrics(ctx)
}
