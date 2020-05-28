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
