//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan_kv_client

import (
	kv "github.com/jfsmig/object-storage/pkg/kv-proto"
	"google.golang.org/grpc"

	"context"
	"time"
)

type ListItem struct {
	Key     string
	Version uint64
}

type Client interface {
	Status(ctx context.Context) (Stats, error)
	Health(ctx context.Context) (string, error)

	Put(ctx context.Context, base, key, value string) error
	Get(ctx context.Context, base, key string) (string, error)
	Delete(ctx context.Context, base, key string) error
	List(ctx context.Context, base, marker string) ([]ListItem, error)
}

type Stats struct {
	B_in     uint64 `json:"b_in"`
	B_out    uint64 `json:"b_out"`
	T_info   uint64 `json:"t_info"`
	T_health uint64 `json:"t_health"`
	T_status uint64 `json:"t_status"`
	T_put    uint64 `json:"t_put"`
	T_get    uint64 `json:"t_get"`
	T_delete uint64 `json:"t_delete"`
	T_list   uint64 `json:"t_list"`
	H_info   uint64 `json:"h_info"`
	H_health uint64 `json:"h_health"`
	H_status uint64 `json:"h_status"`
	H_put    uint64 `json:"h_put"`
	H_get    uint64 `json:"h_get"`
	H_delete uint64 `json:"h_delete"`
	H_list   uint64 `json:"h_list"`
	C_200    uint64 `json:"c_200"`
	C_400    uint64 `json:"c_400"`
	C_404    uint64 `json:"c_404"`
	C_409    uint64 `json:"c_409"`
	C_50X    uint64 `json:"c_50X"`
}

func Dial(url string) (Client, error) {
	cnx, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &grpcClient{cnx: cnx}, err
}

type grpcClient struct {
	cnx *grpc.ClientConn
}

func (self *grpcClient) Get(ctx context.Context, base, key string) (string, error) {
	client := kv.NewKVClient(self.cnx)
	req := kv.GetRequest{Base: base, Key: key, Version: 0}
	rep, err := client.Get(ctx, &req)
	if err != nil {
		return "", err
	}

	return rep.Value, nil
}

func (self *grpcClient) Health(ctx context.Context) (string, error) {
	client := kv.NewKVClient(self.cnx)
	rep, err := client.Health(ctx, &kv.None{})
	if err != nil {
		return "", nil
	}
	return rep.Message, err
}

func (self *grpcClient) List(ctx context.Context, base, marker string) ([]ListItem, error) {
	client := kv.NewKVClient(self.cnx)
	req := kv.ListRequest{Base: base, Marker: marker, MarkerVersion: 0, Max: 1000}
	rep, err := client.List(ctx, &req)
	if err != nil {
		return []ListItem{}, err
	}

	rc := make([]ListItem, 0)
	for _, i := range rep.Items {
		rc = append(rc, ListItem{Key: i.Key, Version: i.Version})
	}
	return rc, err
}

func (self *grpcClient) Put(ctx context.Context, base, key string, value string) error {
	client := kv.NewKVClient(self.cnx)
	req := kv.PutRequest{Base: base, Key: key, Version: uint64(time.Now().UnixNano()), Value: value}
	_, err := client.Put(ctx, &req)
	return err
}

func (self *grpcClient) Delete(ctx context.Context, base, key string) error {
	client := kv.NewKVClient(self.cnx)
	req := kv.DeleteRequest{Base: base, Key: key, Version: uint64(time.Now().UnixNano())}
	_, err := client.Delete(ctx, &req)
	return err
}

func (self *grpcClient) Status(ctx context.Context) (Stats, error) {
	var st Stats
	client := kv.NewKVClient(self.cnx)
	st0, err := client.Status(ctx, &kv.None{})
	if err == nil {
		st = Stats{
			B_in:     st0.BIn,
			B_out:    st0.BOut,
			T_info:   st0.TInfo,
			T_status: st0.TStatus,
			T_health: st0.THealth,
			T_put:    st0.TPut,
			T_get:    st0.TGet,
			T_delete: st0.TDelete,
			T_list:   st0.TList,
			H_info:   st0.HInfo,
			H_status: st0.HStatus,
			H_health: st0.HHealth,
			H_put:    st0.HPut,
			H_get:    st0.HGet,
			H_delete: st0.HDelete,
			H_list:   st0.HList,
			C_200:    st0.C_200,
			C_400:    st0.C_400,
			C_404:    st0.C_404,
			C_409:    st0.C_409,
			C_50X:    st0.C_50X,
		}
	}
	return st, err
}
