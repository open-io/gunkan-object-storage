//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package kv

import (
	"log"
	"time"
)

type logClient struct {
	actual Client
}

func (self *logClient) Close() error {
	return self.actual.Close()
}

func (self *logClient) Ping() error {
	pre := time.Now()
	err := self.actual.Ping()
	log.Printf("Ping() t=%v err=%v", time.Since(pre), err)
	return err
}

func (self *logClient) Get(base, key string) ([]byte, error) {
	pre := time.Now()
	rc, err := self.actual.Get(base, key)
	log.Printf("Get(%s,%s) t=%v err=%v len=%d", base, key, time.Since(pre), err, len(rc))
	return rc, err
}

func (self *logClient) Put(base, key string, value []byte) error {
	pre := time.Now()
	err := self.actual.Put(base, key, value)
	log.Printf("Put(%s,%s,%d) t=%v err=%v", base, key, len(value), time.Since(pre), err)
	return err
}

func (self *logClient) List(base, marker string) ([]ListItem, error) {
	pre := time.Now()
	rc, err := self.actual.List(base, marker)
	log.Printf("List(%s,%s) t=%v err=%v count=%d", base, marker, time.Since(pre), err, len(rc))
	return rc, err
}
