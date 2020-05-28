// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"github.com/jfsmig/object-storage/internal/helpers-grpc"
	kv "github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"google.golang.org/grpc"

	"context"
)

func DialIndexGrpc(url, dirConfig string) (IndexClient, error) {
	cnx, err := helpers_grpc.DialTLSInsecure(url)
	if err != nil {
		return nil, err
	}
	return &IndexGrpcClient{cnx: cnx}, err
}

type IndexGrpcClient struct {
	cnx *grpc.ClientConn
}

func (self *IndexGrpcClient) Get(ctx context.Context, key BaseKey) (string, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.GetRequest{Base: key.Base, Key: key.Key}
	rep, err := client.Get(ctx, &req)
	if err != nil {
		return "", err
	}

	return rep.Value, nil
}

func (self *IndexGrpcClient) List(ctx context.Context, key BaseKey, max uint32) ([]string, error) {
	client := kv.NewIndexClient(self.cnx)
	req := kv.ListRequest{Base: key.Base, Marker: key.Key, Max: max}
	rep, err := client.List(ctx, &req)
	if err != nil {
		return []string{}, err
	}

	return rep.Items, err
}

func (self *IndexGrpcClient) Put(ctx context.Context, key BaseKey, value string) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.PutRequest{Base: key.Base, Key: key.Key, Value: value}
	_, err := client.Put(ctx, &req)
	return err
}

func (self *IndexGrpcClient) Delete(ctx context.Context, key BaseKey) error {
	client := kv.NewIndexClient(self.cnx)
	req := kv.DeleteRequest{Base: key.Base, Key: key.Key}
	_, err := client.Delete(ctx, &req)
	return err
}
