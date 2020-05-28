// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
	"net/http"
)

type HttpSimpleClient struct {
	Endpoint  string
	UserAgent string
	Http      http.Client
}

type HttpMonitorClient interface {
	Info(ctx context.Context) ([]byte, error)
	Health(ctx context.Context) ([]byte, error)
	Metrics(ctx context.Context) ([]byte, error)
}
