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
