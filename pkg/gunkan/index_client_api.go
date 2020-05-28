// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
)

type IndexClient interface {
	Put(ctx context.Context, key BaseKey, value string) error

	Get(ctx context.Context, key BaseKey) (string, error)

	Delete(ctx context.Context, key BaseKey) error

	List(ctx context.Context, marker BaseKey, max uint32) ([]string, error)
}
