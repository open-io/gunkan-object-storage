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
