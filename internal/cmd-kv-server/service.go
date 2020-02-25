//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_kv_server

import (
	"bytes"
	"context"
	"errors"
	"github.com/jfsmig/object-storage/pkg/kv-proto"
	"github.com/tecbot/gorocksdb"
	"log"
	"time"
)

type Config struct {
	Uuid         string
	AddrAnnounce string
	BaseDir      string

	DelayIoError   time.Duration
	DelayFullError time.Duration
}

type Service struct {
	cfg   Config
	db *gorocksdb.DB
}

func NewService(cfg Config) (*Service, error) {
	options := gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(options, cfg.BaseDir)
	if err != nil {
		return nil, err
	}
	srv := Service{cfg: cfg, db: db}
	return &srv, nil
}

func (srv *Service) Health(context.Context, *gunkan_kv_proto.None) (*gunkan_kv_proto.HealthReply, error) {
	return nil, errors.New("NYI")
}

func (srv *Service) Status(context.Context, *gunkan_kv_proto.None) (*gunkan_kv_proto.StatusReply, error) {
	return nil, errors.New("NYI")
}

func (srv *Service) Put(ctx context.Context, req *gunkan_kv_proto.PutRequest) (*gunkan_kv_proto.None, error) {
	var key KeyVersion

	if req.Version == 0 {
		key = Key(req.Base, req.Key, uint64(time.Now().UnixNano()))
	} else {
		// TODO(jfs): dangerous possible override
		key = Key(req.Base, req.Key, req.Version)
	}
	key.Active = true
	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	err := srv.db.Put(opts, encoded, []byte(req.Value))
	if err != nil {
		return nil, err
	} else {
		return &gunkan_kv_proto.None{}, nil
	}
}

func (srv *Service) Delete(ctx context.Context, req *gunkan_kv_proto.DeleteRequest) (*gunkan_kv_proto.None, error) {
	var key KeyVersion

	if req.Version == 0 {
		key = Key(req.Base, req.Key, uint64(time.Now().UnixNano()))
	} else {
		key = Key(req.Base, req.Key, req.Version)
	}
	key.Active = false
	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	err := srv.db.Put(opts, encoded, []byte{})
	if err != nil {
		return nil, err
	} else {
		return &gunkan_kv_proto.None{}, nil
	}
}

func (srv *Service) Get(ctx context.Context, req *gunkan_kv_proto.GetRequest) (*gunkan_kv_proto.GetReply, error) {
	var needle KeyVersion

	if req.Version == 0 {
		needle = KeyLatest(req.Base, req.Key)
	} else {
		needle = Key(req.Base, req.Key, req.Version)
	}
	encoded := []byte(needle.Encode())

	opts := gorocksdb.NewDefaultReadOptions()
	iterator := srv.db.NewIterator(opts)
	iterator.Seek(encoded)
	if !iterator.Valid() {
		return nil, errors.New("Not found")
	}

	var got KeyVersion
	sk := iterator.Key()
	if err := got.DecodeBytes(sk.Data()); err != nil {
		return nil, err
	}

	// Latest item wanted
	if got.Key != needle.Key {
		return nil, errors.New("Not found")
	}
	if !got.Active {
		return nil, errors.New("Deleted")
	}

	rep := gunkan_kv_proto.GetReply{
		Version: got.Version,
		Value:   string(iterator.Value().Data()),
	}
	return &rep, nil
}

func (srv *Service) List(ctx context.Context, req *gunkan_kv_proto.ListRequest) (*gunkan_kv_proto.ListReply, error) {
	if req.Max < 0 {
		req.Max = 1
	} else if req.Max > 1000 {
		req.Max = 1000
	}

	var needle []byte
	if len(req.Base) > 0 {
		if len(req.Marker) > 0 {
			var key KeyVersion
			if req.MarkerVersion == 0 {
				key = KeyLatest(req.Base, req.Marker)
			} else {
				key = Key(req.Base, req.Marker, req.MarkerVersion)
			}
			needle = []byte(key.Encode())
		} else {
			needle = []byte(req.Base + ",")
		}
	}

	opts := gorocksdb.NewDefaultReadOptions()
	iterator := srv.db.NewIterator(opts)
	iterator.Seek(needle)

	rep := gunkan_kv_proto.ListReply{}
	for ; iterator.Valid(); iterator.Next() {
		if bytes.Compare(iterator.Key().Data(), needle) > 0 {
			break
		}
	}
	for ; iterator.Valid(); iterator.Next() {
		log.Println("!", string(iterator.Key().Data()))
		if uint32(len(rep.Items)) > req.Max {
			break
		}
		sk := iterator.Key()
		rep.Items = append(rep.Items, &gunkan_kv_proto.ObjectId{
			Version: 0,
			Key:     string(sk.Data()),
		})
	}
	return &rep, nil
}
