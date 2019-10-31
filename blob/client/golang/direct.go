//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package blob

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type httpClient struct {
	endpoint string
	client   http.Client
}

func newHttpClient(url string) Client {
	return &httpClient{endpoint: url, client: http.Client{}}
}

func (self *httpClient) Close() error {
	self.client.CloseIdleConnections()
	return nil
}

func (self *httpClient) Delete(id Id) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := http.NewRequest("DELETE", b.String(), nil)
	if err != nil {
		return err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 200, 201, 204:
		return nil
	default:
		return ErrInternalError
	}
}

func (self *httpClient) Get(id Id) (io.ReadCloser, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := http.NewRequest("GET", b.String(), nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch rep.StatusCode {
	case 404:
		return nil, ErrNotFound
	case 403:
		return nil, ErrForbidden
	case 200, 201, 204:
		return rep.Body, nil
	default:
		return nil, ErrInternalError
	}
}

func (self *httpClient) PutN(id Id, data io.Reader, size int64) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := http.NewRequest("PUT", b.String(), data)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Length", strconv.FormatInt(size, 10))

	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 409:
		return ErrAlreadyExists
	case 200, 201, 204:
		return nil
	default:
		return ErrInternalError
	}
}

func (self *httpClient) Put(id Id, data io.Reader) error {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/blob/")
	id.EncodeIn(&b)

	req, err := http.NewRequest("PUT", b.String(), data)
	if err != nil {
		return err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 409:
		return ErrAlreadyExists
	case 200, 201, 204:
		return nil
	default:
		return ErrInternalError
	}
}

func (self *httpClient) List(max uint) ([]Id, error) {
	return self.listRaw(max, "")
}

func (self *httpClient) ListAfter(max uint, marker Id) ([]Id, error) {
	return self.listRaw(max, marker.Encode())
}

func (self *httpClient) listRaw(max uint, marker string) ([]Id, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/list")
	if len(marker) > 0 {
		b.WriteRune('/')
		b.WriteString(marker)
	}

	req, err := http.NewRequest("GET", b.String(), nil)
	if err != nil {
		return nil, err
	}

	rep, err := self.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rep.Body.Close()
	switch rep.StatusCode {
	case 404:
		return nil, ErrNotFound
	case 403:
		return nil, ErrForbidden
	case 200, 201, 204:
		return unpackBlobIdArray(rep.Body)
	default:
		return nil, ErrInternalError
	}
}

func unpackBlobIdArray(body io.Reader) ([]Id, error) {
	rc := make([]Id, 0)
	r := bufio.NewReader(body)
	for {
		if line, err := r.ReadString('\n'); err != nil {
			if err == io.EOF {
				return rc, nil
			} else {
				return nil, err
			}
		} else if len(line) > 0 {
			var id Id
			line = strings.Trim(line, "\r\n")
			if err = id.Decode(line); err != nil {
				return nil, err
			} else {
				rc = append(rc, id)
			}
		}
	}
}

func (self *httpClient) Status() (Stats, error) {
	b := strings.Builder{}
	b.WriteString("http://")
	b.WriteString(self.endpoint)
	b.WriteString("/v1/status")

	if req, err := http.NewRequest("GET", b.String(), nil); err != nil {
		return Stats{}, err
	} else if rep, err := self.client.Do(req); err != nil {
		return Stats{}, err
	} else {
		defer rep.Body.Close()
		var st Stats
		err := json.NewDecoder(rep.Body).Decode(&st)
		return st, err
	}
}
