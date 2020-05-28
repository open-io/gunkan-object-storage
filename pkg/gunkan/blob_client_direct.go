// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"strings"
)

type httpBlobClient struct {
	client HttpSimpleClient
}

func DialBlob(url string) (BlobClient, error) {
	var err error
	var rc httpBlobClient
	err = rc.client.Init(url)
	if err != nil {
		return nil, err
	}
	return &rc, nil
}

func (self *httpBlobClient) Delete(ctx context.Context, realid string) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString("/v1/blob/")
	b.WriteString(realid)

	req, err := self.client.makeRequest(ctx, "DELETE", b.String(), nil)
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

func (self *httpBlobClient) Get(ctx context.Context, realid string) (io.ReadCloser, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString("/v1/blob/")
	b.WriteString(realid)

	req, err := self.client.makeRequest(ctx, "GET", b.String(), nil)
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

func (self *httpBlobClient) PutN(ctx context.Context, id BlobId, data io.Reader, size int64) (string, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := self.client.makeRequest(ctx, "PUT", b.String(), data)
	if err != nil {
		return "", err
	}

	req.ContentLength = size
	rep, err := self.client.Http.Do(req)
	if err != nil {
		return "", err
	}

	defer rep.Body.Close()
	return "", MapCodeToError(rep.StatusCode)
}

func (self *httpBlobClient) Put(ctx context.Context, id BlobId, data io.Reader) (string, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := self.client.makeRequest(ctx, "PUT", b.String(), data)
	if err != nil {
		return "", err
	}

	req.ContentLength = -1
	rep, err := self.client.Http.Do(req)
	if err != nil {
		return "", err
	}

	defer rep.Body.Close()
	return "", MapCodeToError(rep.StatusCode)
}

func (self *httpBlobClient) List(ctx context.Context, max uint) ([]BlobListItem, error) {
	return self.listRaw(ctx, max, "")
}

func (self *httpBlobClient) ListAfter(ctx context.Context, max uint, marker string) ([]BlobListItem, error) {
	return self.listRaw(ctx, max, marker)
}

func (self *httpBlobClient) listRaw(ctx context.Context, max uint, marker string) ([]BlobListItem, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString("/v1/list")
	if len(marker) > 0 {
		b.WriteRune('/')
		b.WriteString(marker)
	}

	req, err := self.client.makeRequest(ctx, "GET", b.String(), nil)
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
		return unpackBlobIdArray(rep.Body)
	default:
		return nil, MapCodeToError(rep.StatusCode)
	}
}

func unpackBlobIdArray(body io.Reader) ([]BlobListItem, error) {
	rc := make([]BlobListItem, 0)
	r := bufio.NewReader(body)
	for {
		if line, err := r.ReadString('\n'); err != nil {
			if err == io.EOF {
				return rc, nil
			} else {
				return nil, err
			}
		} else if len(line) > 0 {
			var id BlobId
			line = strings.Trim(line, "\r\n")
			if id, err = DecodeBlobId(line); err != nil {
				return nil, err
			} else {
				rc = append(rc, BlobListItem{"", id})
			}
		}
	}
}

func (self *httpBlobClient) srvGet(ctx context.Context, tag string) ([]byte, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.client.Endpoint)
	b.WriteString(tag)

	req, err := self.client.makeRequest(ctx, "GET", b.String(), nil)
	if err != nil {
		return []byte{}, err
	}

	rep, err := self.client.Http.Do(req)
	if err != nil {
		return []byte{}, err
	}

	defer rep.Body.Close()
	return ioutil.ReadAll(rep.Body)
}

func (self *httpBlobClient) Info(ctx context.Context) ([]byte, error) {
	return self.client.Info(ctx)
}

func (self *httpBlobClient) Health(ctx context.Context) ([]byte, error) {
	return self.client.Health(ctx)
}

func (self *httpBlobClient) Metrics(ctx context.Context) ([]byte, error) {
	return self.client.Metrics(ctx)
}
