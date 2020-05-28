// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
	"io"
)

type PartClient interface {
	Put(ctx context.Context, id PartId, data io.Reader) error
	PutN(ctx context.Context, id PartId, data io.Reader, size int64) error

	Get(ctx context.Context, id PartId) (io.ReadCloser, error)

	Delete(ctx context.Context, id PartId) error

	// Returns the first page of known parts in the current pod
	List(ctx context.Context, max uint32) ([]PartId, error)

	// Returns the next page of known parts in the current pod
	ListAfter(ctx context.Context, max uint32, id PartId) ([]PartId, error)
}
