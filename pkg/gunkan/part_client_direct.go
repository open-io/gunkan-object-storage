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

type httpPartClient struct {
	endpoint string
	client   http.Client
}

func DialPart(url string) (PartClient, error) {
	return newHttpPartClient(url), nil
}

func newHttpPartClient(url string) PartClient {
	return &httpPartClient{endpoint: url, client: http.Client{}}
}

func (self *httpPartClient) Close() error {
	self.client.CloseIdleConnections()
	return nil
}

func (self *httpPartClient) Delete(ctx context.Context, id PartId) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

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

func (self *httpPartClient) Get(ctx context.Context, id PartId) (io.ReadCloser, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

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

func (self *httpPartClient) PutN(ctx context.Context, id PartId, data io.Reader, size int64) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := makeRequest("PUT", b.String(), data)
	if err != nil {
		return err
	}

	req.ContentLength = size
	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return codeMapper(rep.StatusCode)
}

func (self *httpPartClient) Put(ctx context.Context, id PartId, data io.Reader) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := makeRequest("PUT", b.String(), data)
	if err != nil {
		return err
	}

	req.ContentLength = -1
	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	return codeMapper(rep.StatusCode)
}

func (self *httpPartClient) List(ctx context.Context, max uint) ([]PartId, error) {
	return self.listRaw(max, "")
}

func (self *httpPartClient) ListAfter(ctx context.Context, max uint, id PartId) ([]PartId, error) {
	return self.listRaw(max, id.EncodeMarker())
}

func (self *httpPartClient) Status(ctx context.Context) (PartStats, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/status")

	req, err := makeRequest("GET", b.String(), nil)
	if err != nil {
		return PartStats{}, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return PartStats{}, err
	}

	defer rep.Body.Close()
	var st PartStats
	err = json.NewDecoder(rep.Body).Decode(&st)
	return st, err
}

func (self *httpPartClient) Health(ctx context.Context) (string, error) {
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

func (self *httpPartClient) listRaw(max uint, marker string) ([]PartId, error) {
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
		return unpackPartIdArray(rep.Body)
	default:
		return nil, codeMapper(rep.StatusCode)
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
