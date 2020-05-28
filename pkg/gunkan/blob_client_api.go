// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

import (
	"context"
	"io"
)

type BlobClient interface {
	Put(ctx context.Context, id BlobId, data io.Reader) (string, error)
	PutN(ctx context.Context, id BlobId, data io.Reader, size int64) (string, error)

	Get(ctx context.Context, realId string) (io.ReadCloser, error)

	Delete(ctx context.Context, realId string) error

	List(ctx context.Context, max uint) ([]BlobListItem, error)
	ListAfter(ctx context.Context, max uint, marker string) ([]BlobListItem, error)
}

type BlobListItem struct {
	Real    string
	Logical BlobId
}
