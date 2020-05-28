// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_store_fs

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math"
	"time"
)

type config struct {
	uuid         string
	addrBind     string
	addrAnnounce string
	dirConfig    string
	dirBase      string

	delayIoError   time.Duration
	delayFullError time.Duration
}

type service struct {
	config config

	repo Repo

	lastIoError   time.Time
	lastFullError time.Time

	timePut  prometheus.Histogram
	timeGet  prometheus.Histogram
	timeDel  prometheus.Histogram
	timeList prometheus.Histogram
}

func newService(cfg config) (*service, error) {
	var err error
	srv := service{config: cfg}

	srv.repo, err = MakePostNamed(cfg.dirBase)
	if err != nil {
		return nil, err
	}

	buckets := []float64{0.01, 0.02, 0.03, 0.04, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1, 2, 3, 4, 5, math.Inf(1)}

	srv.timeList = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gunkan_blob_list_ttlb",
		Help:    "Repartition of the request times of List requests",
		Buckets: buckets,
	})

	srv.timePut = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gunkan_blob_put_ttlb",
		Help:    "Repartition of the request times of put requests",
		Buckets: buckets,
	})

	srv.timeGet = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gunkan_blob_get_ttlb",
		Help:    "Repartition of the request times of get requests",
		Buckets: buckets,
	})

	srv.timeDel = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gunkan_blob_del_ttlb",
		Help:    "Repartition of the request times of del requests",
		Buckets: buckets,
	})

	if err != nil {
		return nil, err
	} else {
		return &srv, nil
	}
}

func (srv *service) isFull(now time.Time) bool {
	return !srv.lastFullError.IsZero() && now.Sub(srv.lastFullError) > srv.config.delayFullError
}

func (srv *service) isError(now time.Time) bool {
	return !srv.lastIoError.IsZero() && now.Sub(srv.lastIoError) > srv.config.delayIoError
}

func (srv *service) isOverloaded(now time.Time) bool {
	return false
}
