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
	helpers_grpc "github.com/jfsmig/object-storage/internal/helpers-grpc"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	proto "github.com/jfsmig/object-storage/pkg/gunkan-index-proto"
	"google.golang.org/grpc"
	"sync"
	"time"
)

const (
	parallelismPut    = 5
	parallelismGet    = 5
	parallelismDelete = 5
	parallelismList   = 5
)

type serviceConfig struct {
	uuid         string
	addrBind     string
	addrAnnounce string
	dirConfig    string
}

type service struct {
	cfg          serviceConfig
	discovery    gunkan.Discovery
	wg           sync.WaitGroup
	rw           sync.RWMutex
	back         map[string]*grpc.ClientConn
	flag_running bool
}

func NewService(config serviceConfig) (*service, error) {
	srv := service{}
	srv.cfg = config
	srv.flag_running = true
	srv.back = make(map[string]*grpc.ClientConn)

	var err error
	srv.discovery, err = gunkan.NewDiscoveryDefault()
	if err != nil {
		return nil, err
	}

	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		for srv.flag_running {
			tick := time.After(1 * time.Second)
			<-tick
			srv.reload()
		}
	}()
	return &srv, nil
}

func (srv *service) reload() {
	srv.rw.Lock()
	defer srv.rw.Unlock()

	// Get all the declared backends
	addrs, err := srv.discovery.ListIndexStore()
	if err != nil {
		gunkan.Logger.Warn().Err(err).Msg("Discovery: index stores")
		return
	}

	// Open a connection to each new declared backend.
	// We avoid closing/reopening connections to stable backends
	for _, a := range addrs {
		if c, ok := srv.back[a]; ok && c != nil {
			continue
		}
		c, err := helpers_grpc.DialTLSInsecure(a)
		if err != nil {
			gunkan.Logger.Warn().Err(err).Str("to", a).Msg("Connection error to index")
			srv.back[a] = nil
		} else {
			srv.back[a] = c
		}
	}
}

func (srv *service) Join() {
	srv.flag_running = false
	srv.wg.Wait()
}

type targetError struct {
	addr string
	err  error
}

type targetErrorValue struct {
	targetError
	value   string
	version uint64
}

type targetErrorList struct {
	targetError
	items []string
}

type targetInput struct {
	addr string
	cnx  *grpc.ClientConn
}

func mergeTargetError(chans ...<-chan targetError) <-chan targetError {
	var wg sync.WaitGroup
	out := make(chan targetError)
	consume := func(input <-chan targetError) {
		for i := range input {
			out <- i
		}
		wg.Done()
	}

	wg.Add(len(chans))
	for _, c := range chans {
		go consume(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func mergeTargetValueError(chans ...<-chan targetErrorValue) <-chan targetErrorValue {
	var wg sync.WaitGroup
	out := make(chan targetErrorValue)
	consume := func(input <-chan targetErrorValue) {
		for i := range input {
			out <- i
		}
		wg.Done()
	}

	wg.Add(len(chans))
	for _, c := range chans {
		go consume(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func mergeTargetValueList(chans ...<-chan targetErrorList) <-chan targetErrorList {
	var wg sync.WaitGroup
	out := make(chan targetErrorList)
	consume := func(input <-chan targetErrorList) {
		for i := range input {
			out <- i
		}
		wg.Done()
	}

	wg.Add(len(chans))
	for _, c := range chans {
		go consume(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (srv *service) Put(ctx context.Context, req *proto.PutRequest) (*proto.None, error) {
	work := func(input <-chan targetInput) <-chan targetError {
		out := make(chan targetError, 1)
		go func() {
			for i := range input {
				cli := proto.NewIndexClient(i.cnx)
				_, err := cli.Put(ctx, req)
				out <- targetError{i.addr, err}
			}
			close(out)
		}()
		return out
	}

	srv.rw.RLock()
	defer srv.rw.RUnlock()

	in := make(chan targetInput, len(srv.back))
	outv := make([]<-chan targetError, 0)
	for i := 0; i < parallelismPut; i++ {
		outv = append(outv, work(in))
	}
	out := mergeTargetError(outv...)

	for addr, cnx := range srv.back {
		in <- targetInput{addr: addr, cnx: cnx}
	}
	close(in)
	any := false
	for err := range out {
		if err.err == nil {
			gunkan.Logger.Debug().
				Str("op", "PUT").Str("k", req.Key).Str("srv", err.addr)
			any = true
		} else {
			gunkan.Logger.Warn().
				Str("op", "PUT").Str("k", req.Key).Str("srv", err.addr).Err(err.err)
		}
	}

	if !any {
		return nil, errors.New("No backend replied")
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.None, error) {
	work := func(input <-chan targetInput) <-chan targetError {
		out := make(chan targetError)
		go func() {
			for i := range input {
				cli := proto.NewIndexClient(i.cnx)
				_, err := cli.Delete(ctx, req)
				out <- targetError{i.addr, err}
			}
			close(out)
		}()
		return out
	}

	srv.rw.RLock()
	in := make(chan targetInput, len(srv.back))
	outv := make([]<-chan targetError, 0)
	for i := 0; i < parallelismDelete; i++ {
		outv = append(outv, work(in))
	}
	out := mergeTargetError(outv...)
	for addr, cnx := range srv.back {
		in <- targetInput{addr: addr, cnx: cnx}
	}
	close(in)
	any := false
	for err := range out {
		if err.err == nil {
			gunkan.Logger.Debug().
				Str("op", "DEL").Str("k", req.Key).Str("srv", err.addr)
			any = true
		} else {
			gunkan.Logger.Debug().
				Str("op", "DEL").Str("k", req.Key).Str("srv", err.addr).Err(err.err)
		}
	}
	srv.rw.RUnlock()

	if !any {
		return nil, errors.New("No backend replied")
	} else {
		return &proto.None{}, nil
	}
}

func (srv *service) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetReply, error) {
	work := func(input <-chan targetInput) <-chan targetErrorValue {
		out := make(chan targetErrorValue, 1)
		go func() {
			for i := range input {
				cli := proto.NewIndexClient(i.cnx)
				rep, err := cli.Get(ctx, req)
				rc := targetErrorValue{}
				rc.addr = i.addr
				rc.err = err
				if err == nil {
					rc.value = rep.Value
					rc.version = rep.Version
				}
				out <- rc
			}
			close(out)
		}()
		return out
	}

	srv.rw.RLock()
	srv.rw.RUnlock()

	in := make(chan targetInput, len(srv.back))
	outv := make([]<-chan targetErrorValue, 0)
	for i := 0; i < parallelismGet; i++ {
		outv = append(outv, work(in))
	}
	out := mergeTargetValueError(outv...)

	for addr, cnx := range srv.back {
		in <- targetInput{addr: addr, cnx: cnx}
	}
	close(in)

	any := false
	rep := proto.GetReply{Value: "", Version: 0}
	for x := range out {
		if x.err != nil {
			gunkan.Logger.Warn().Str("op", "GET").Str("k", req.Key).Str("srv", x.addr).Err(x.err)
		} else {
			any = true
			if x.version > rep.Version {
				rep.Version = x.version
				rep.Value = x.value
			}
		}
	}

	if !any {
		return nil, errors.New("No backend replied")
	} else {
		return &rep, nil
	}
}

func (srv *service) List(ctx context.Context, req *proto.ListRequest) (*proto.ListReply, error) {
	work := func(input <-chan targetInput) <-chan targetErrorList {
		out := make(chan targetErrorList, 1)
		go func() {
			for i := range input {
				cli := proto.NewIndexClient(i.cnx)
				rep, err := cli.List(ctx, req)
				rc := targetErrorList{}
				rc.addr = i.addr
				rc.err = err
				if err == nil {
					rc.items = rep.Items[:]
				}
				out <- rc
			}
			close(out)
		}()
		return out
	}

	if req.Max <= 0 {
		req.Max = 1
	} else if req.Max > gunkan.ListHardMax {
		req.Max = gunkan.ListHardMax
	}

	srv.rw.RLock()
	srv.rw.RUnlock()

	in := make(chan targetInput, len(srv.back))
	outv := make([]<-chan targetErrorList, 0)
	for i := 0; i < parallelismList; i++ {
		outv = append(outv, work(in))
	}
	out := mergeTargetValueList(outv...)

	for addr, cnx := range srv.back {
		in <- targetInput{addr: addr, cnx: cnx}
	}
	close(in)

	any := false
	rep := proto.ListReply{}

	tabs := make([][]string, 0)
	for x := range out {
		if x.err != nil {
			gunkan.Logger.Info().
				Str("op", "LIST").
				Str("k", req.Marker).Str("srv", x.addr).
				Err(x.err)
		} else {
			any = true
			tabs = append(tabs, x.items)
		}
	}

	for x := range keepSingleDedup(tabs, req.Max) {
		rep.Items = append(rep.Items, x)
	}
	if !any {
		return nil, errors.New("No backend replied")
	} else {
		return &rep, nil
	}
}

func keepSingleDedup(tabs [][]string, max uint32) <-chan string {
	nbTabs := len(tabs)
	maxes := make([]int, nbTabs, nbTabs)
	indices := make([]int, nbTabs, nbTabs)
	for i, tab := range tabs {
		maxes[i] = len(tab)
	}
	out := make(chan string, 1024)

	min := func() string {
		var min string
		var iMin int
		for i, tab := range tabs {
			if indices[i] >= maxes[i] {
				continue
			}
			obj := tab[indices[i]]
			if min == "" || obj < min {
				iMin = i
				min = obj
			}
		}
		if min != "" {
			indices[iMin]++
		}
		return min
	}

	var last string
	poll := func() string {
		for {
			obj := min()
			if obj == "" {
				return ""
			}
			if last == "" {
				last = obj
				return last
			}
			if last == obj {
				continue
			}
			last = obj
			return last
		}
	}

	go func() {
		sent := uint32(0)
		for obj := poll(); obj != ""; obj = poll() {
			if sent >= max {
				break
			}
			out <- obj
			sent++
		}
		close(out)
	}()

	return out
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
