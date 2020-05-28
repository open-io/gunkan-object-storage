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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	key := gunkan.BK(req.Base, req.Key)

	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	defer opts.Destroy()
	opts.SetSync(false)
	err := srv.db.Put(opts, encoded, []byte(req.Value))
	if err != nil {
		return nil, err
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.None, error) {
	key := gunkan.BK(req.Base, req.Key)
	encoded := []byte(key.Encode())

	// FIXME(jfs): check if the KV is present

	opts := gorocksdb.NewDefaultWriteOptions()
	opts.SetSync(false)
	defer opts.Destroy()
	err := srv.db.Put(opts, encoded, []byte{})
	if err != nil {
		return nil, err
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetReply, error) {
	needle := gunkan.BK(req.Base, req.Key)
	encoded := []byte(needle.Encode())

	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	opts.SetFillCache(true)
	iterator := srv.db.NewIterator(opts)
	iterator.Seek(encoded)
	if !iterator.Valid() {
		return nil, errors.New("Not found")
	}

	var got gunkan.BaseKey
	sk := iterator.Key()
	if err := got.DecodeBytes(sk.Data()); err != nil {
		return nil, err
	}

	// Latest item wanted
	if got.Base != needle.Base || got.Key != needle.Key {
		return nil, errors.New("Not found")
	}

	return &proto.GetReply{Value: string(iterator.Value().Data())}, nil
}

func (srv *service) List(ctx context.Context, req *proto.ListRequest) (*proto.ListReply, error) {
	if req.Max < 0 {
		req.Max = 1
	} else if req.Max > gunkan.ListHardMax {
		req.Max = gunkan.ListHardMax
	}

	if req.Base == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing base")
	}

	var needle []byte
	if len(req.Base) > 0 {
		if len(req.Marker) > 0 {
			needle = []byte(gunkan.BK(req.Base, req.Marker).Encode())
		} else {
			needle = []byte(req.Base + ",")
		}
	}

	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	opts.SetFillCache(true)
	iterator := srv.db.NewIterator(opts)

	rep := proto.ListReply{}

	iterator.Seek(needle)
	for ; iterator.Valid(); iterator.Next() {
		if bytes.Compare(iterator.Key().Data(), needle) > 0 {
			break
		}
	}
	for ; iterator.Valid(); iterator.Next() {
		// Check we didn't reach the max elements
		if uint32(len(rep.Items)) > req.Max {
			break
		}

		// Check the base matches
		sk := iterator.Key()
		var k gunkan.BaseKey
		err := k.DecodeBytes(sk.Data())
		if err != nil {
			return nil, status.Errorf(codes.DataLoss, "Malformed DB entry")
		}
		if k.Base != req.Base {
			break
		}

		rep.Items = append(rep.Items, k.Key)
	}

	return &rep, nil
}
