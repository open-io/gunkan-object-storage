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
	"context"
	"io/ioutil"

	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type httpBlobClient struct {
	endpoint string
	client   http.Client
}

func DialBlob(url string) (BlobClient, error) {
	return newHttpBlobClient(url), nil
}

func newHttpBlobClient(url string) BlobClient {
	return &httpBlobClient{endpoint: url, client: http.Client{}}
}

func (self *httpBlobClient) Close() error {
	self.client.CloseIdleConnections()
	return nil
}

func (self *httpBlobClient) Delete(ctx context.Context, realid string) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	b.WriteString(realid)

	req, err := makeRequest("DELETE", b.String(), nil)
	if err != nil {
		return err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return codeMapper(rep.StatusCode)
}

func (self *httpBlobClient) Get(ctx context.Context, realid string) (io.ReadCloser, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	b.WriteString(realid)

	req, err := makeRequest("GET", b.String(), nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch rep.StatusCode {
	case 200, 201, 204:
		return rep.Body, nil
	default:
		return nil, codeMapper(rep.StatusCode)
	}
}

func (self *httpBlobClient) PutN(ctx context.Context, id BlobId, data io.Reader, size int64) (string, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := makeRequest("PUT", b.String(), data)
	if err != nil {
		return "", err
	}

	req.ContentLength = size
	rep, err := self.client.Do(req)
	if err != nil {
		return "", err
	}

	defer rep.Body.Close()
	return "", codeMapper(rep.StatusCode)
}

func (self *httpBlobClient) Put(ctx context.Context, id BlobId, data io.Reader) (string, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := makeRequest("PUT", b.String(), data)
	if err != nil {
		return "", err
	}

	req.ContentLength = -1
	rep, err := self.client.Do(req)
	if err != nil {
		return "", err
	}

	defer rep.Body.Close()
	return "", codeMapper(rep.StatusCode)
}

func (self *httpBlobClient) List(ctx context.Context, max uint) ([]BlobListItem, error) {
	return self.listRaw(max, "")
}

func (self *httpBlobClient) ListAfter(ctx context.Context, max uint, marker string) ([]BlobListItem, error) {
	return self.listRaw(max, marker)
}

func (self *httpBlobClient) listRaw(max uint, marker string) ([]BlobListItem, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/list")
	if len(marker) > 0 {
		b.WriteRune('/')
		b.WriteString(marker)
	}

	req, err := makeRequest("GET", b.String(), nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 200, 201, 204:
		return unpackBlobIdArray(rep.Body)
	default:
		return nil, codeMapper(rep.StatusCode)
	}
}

func (self *httpBlobClient) Status(ctx context.Context) (Stats, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/status")

	req, err := makeRequest("GET", b.String(), nil)
	if err != nil {
		return Stats{}, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return Stats{}, err
	}

	defer rep.Body.Close()
	var st Stats
	err = json.NewDecoder(rep.Body).Decode(&st)
	return st, err
}

func (self *httpBlobClient) Health(ctx context.Context) (string, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/health")

	req, err := makeRequest("GET", b.String(), nil)
	if err != nil {
		return "", err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return "", err
	}

	defer rep.Body.Close()
	body, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
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
			if err = id.Decode(line); err != nil {
				return nil, err
			} else {
				rc = append(rc, BlobListItem{"", id})
			}
		}
	}
}
