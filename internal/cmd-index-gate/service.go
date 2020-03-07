//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_index_gate

import (
	"context"
	"errors"
	proto "github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
)

type serviceConfig struct {
	Uuid         string
	AddrBind     string
	AddrAnnounce string
	DirConfig    string
}

type service struct {
	cfg serviceConfig
}

func NewService() (*service, error) {
	srv := service{}
	return &srv, nil
}

func (srv *service) Put(ctx context.Context, req *proto.PutRequest) (*proto.None, error) {
	return nil, errors.New("NYI")
}

func (srv *service) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.None, error) {
	return nil, errors.New("NYI")
}

func (srv *service) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetReply, error) {
	return nil, errors.New("NYI")
}

func (srv *service) List(ctx context.Context, req *proto.ListRequest) (*proto.ListReply, error) {
	return nil, errors.New("NYI")
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
