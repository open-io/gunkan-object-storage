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
	"github.com/jfsmig/object-storage/internal/helpers-grpc"
	kv "github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"google.golang.org/grpc"

	"context"
	"errors"
	"time"
)

func DialIndex(url, dirConfig string) (IndexClient, error) {
	cnx, err := helpers_grpc.DialTLS(url, dirConfig)
	if err != nil {
		return nil, err
	}
	return &grpcClient{cnx: cnx}, err
}

type grpcClient struct {
	cnx *grpc.ClientConn
}

func (self *grpcClient) Get(ctx context.Context, base, key string) (string, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.GetRequest{Base: base, Key: key, Version: 0}
	rep, err := client.Get(ctx, &req)
	if err != nil {
		return "", err
	}

	return rep.Value, nil
}

func (self *grpcClient) List(ctx context.Context, base, marker string, max uint32) ([]IndexListItem, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.ListRequest{Base: base, Marker: marker, MarkerVersion: 0, Max: max}
	rep, err := client.List(ctx, &req)
	if err != nil {
		return []IndexListItem{}, err
	}

	rc := make([]IndexListItem, 0)
	for _, i := range rep.Items {
		rc = append(rc, IndexListItem{Key: i.Key, Version: i.Version})
	}
	return rc, err
}

func (self *grpcClient) Put(ctx context.Context, base, key string, value string) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.PutRequest{Base: base, Key: key, Version: uint64(time.Now().UnixNano()), Value: value}
	_, err := client.Put(ctx, &req)
	return err
}

func (self *grpcClient) Delete(ctx context.Context, base, key string) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.DeleteRequest{Base: base, Key: key, Version: uint64(time.Now().UnixNano())}
	_, err := client.Delete(ctx, &req)
	return err
}

func (self *grpcClient) Status(ctx context.Context) (IndexStats, error) {
	// FIXME(jfs): Query using HTTP
	return IndexStats{}, errors.New("NYI")
}

func (self *grpcClient) Health(ctx context.Context) (string, error) {
	// FIXME(jfs): Query using HTTP
	return "", errors.New("NYI")
}
