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
)

func DialIndex(url, dirConfig string) (IndexClient, error) {
	cnx, err := helpers_grpc.DialTLSInsecure(url)
	if err != nil {
		return nil, err
	}
	return &IndexGrpcClient{cnx: cnx}, err
}

type IndexGrpcClient struct {
	cnx *grpc.ClientConn
}

func (self *IndexGrpcClient) Get(ctx context.Context, key BaseKeyVersion) (string, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.GetRequest{Base: key.Base, Key: key.Key, Version: 0}
	rep, err := client.Get(ctx, &req)
	if err != nil {
		return "", err
	}

	return rep.Value, nil
}

func (self *IndexGrpcClient) List(ctx context.Context, key BaseKeyVersion, max uint32) ([]KeyVersion, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.ListRequest{Base: key.Base, Marker: key.Key, MarkerVersion: 0, Max: max}
	rep, err := client.List(ctx, &req)
	if err != nil {
		return []KeyVersion{}, err
	}

	rc := make([]KeyVersion, 0)
	for _, i := range rep.Items {
		rc = append(rc, KeyVersion{Key: i.Key, Version: i.Version})
	}
	return rc, err
}

func (self *IndexGrpcClient) Put(ctx context.Context, key BaseKeyVersion, value string) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.PutRequest{Base: key.Base, Key: key.Key, Version: key.Version, Value: value}
	_, err := client.Put(ctx, &req)
	return err
}

func (self *IndexGrpcClient) Delete(ctx context.Context, key BaseKeyVersion) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.DeleteRequest{Base: key.Base, Key: key.Key, Version: key.Version}
	_, err := client.Delete(ctx, &req)
	return err
}

func (self *IndexGrpcClient) Status(ctx context.Context) (IndexStats, error) {
	// FIXME(jfs): Query using HTTP
	return IndexStats{}, errors.New("NYI")
}

func (self *IndexGrpcClient) Health(ctx context.Context) (string, error) {
	// FIXME(jfs): Query using HTTP
	return "", errors.New("NYI")
}
