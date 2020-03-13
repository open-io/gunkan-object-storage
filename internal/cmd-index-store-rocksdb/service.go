//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_index_store_rocksdb

import (
	"bytes"
	"context"
	"errors"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	proto "github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"github.com/tecbot/gorocksdb"
	"log"
	"time"
)

type serviceConfig struct {
	uuid         string
	addrBind     string
	addrAnnounce string
	dirConfig    string
	dirBase      string

	delayIoError   time.Duration
	delayFullError time.Duration
}

type service struct {
	cfg serviceConfig
	db  *gorocksdb.DB
}

func NewService(cfg serviceConfig) (*service, error) {
	options := gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(options, cfg.dirBase)
	if err != nil {
		return nil, err
	}
	srv := service{cfg: cfg, db: db}
	return &srv, nil
}

func (srv *service) Put(ctx context.Context, req *proto.PutRequest) (*proto.None, error) {
	var key gunkan.BaseKeyVersion

	if req.Version == 0 {
		key = gunkan.BaseKey(req.Base, req.Key, uint64(time.Now().UnixNano()))
	} else {
		// TODO(jfs): dangerous possible override
		key = gunkan.BaseKey(req.Base, req.Key, req.Version)
	}
	key.Active = true
	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	err := srv.db.Put(opts, encoded, []byte(req.Value))
	if err != nil {
		return nil, err
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.None, error) {
	var key gunkan.BaseKeyVersion

	if req.Version == 0 {
		key = gunkan.BaseKey(req.Base, req.Key, uint64(time.Now().UnixNano()))
	} else {
		key = gunkan.BaseKey(req.Base, req.Key, req.Version)
	}
	key.Active = false
	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	err := srv.db.Put(opts, encoded, []byte{})
	if err != nil {
		return nil, err
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetReply, error) {
	var needle gunkan.BaseKeyVersion

	if req.Version == 0 {
		needle = gunkan.BaseKeyLatest(req.Base, req.Key)
	} else {
		needle = gunkan.BaseKey(req.Base, req.Key, req.Version)
	}
	encoded := []byte(needle.Encode())

	opts := gorocksdb.NewDefaultReadOptions()
	iterator := srv.db.NewIterator(opts)
	iterator.Seek(encoded)
	if !iterator.Valid() {
		return nil, errors.New("Not found")
	}

	var got gunkan.KeyVersion
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

	rep := proto.GetReply{
		Version: got.Version,
		Value:   string(iterator.Value().Data()),
	}
	return &rep, nil
}

func (srv *service) List(ctx context.Context, req *proto.ListRequest) (*proto.ListReply, error) {
	if req.Max < 0 {
		req.Max = 1
	} else if req.Max > 1000 {
		req.Max = 1000
	}

	var needle []byte
	if len(req.Base) > 0 {
		if len(req.Marker) > 0 {
			var key gunkan.BaseKeyVersion
			if req.MarkerVersion == 0 {
				key = gunkan.BaseKeyLatest(req.Base, req.Marker)
			} else {
				key = gunkan.BaseKey(req.Base, req.Marker, req.MarkerVersion)
			}
			needle = []byte(key.Encode())
		} else {
			needle = []byte(req.Base + ",")
		}
	}

	opts := gorocksdb.NewDefaultReadOptions()
	iterator := srv.db.NewIterator(opts)
	iterator.Seek(needle)

	rep := proto.ListReply{}
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
		rep.Items = append(rep.Items, &proto.ObjectId{
			Version: 0,
			Key:     string(sk.Data()),
		})
	}
	return &rep, nil
}

func (srv *service) Info(ctx context.Context, req *proto.None) (*proto.InfoReply, error) {
	return nil, errors.New("NYI")
}

func (srv *service) Health(ctx context.Context, req *proto.None) (*proto.HealthReply, error) {
	return nil, errors.New("NYI")
}

func (srv *service) Stats(ctx context.Context, req *proto.None) (*proto.StatsReply, error) {
	return nil, errors.New("NYI")
}
