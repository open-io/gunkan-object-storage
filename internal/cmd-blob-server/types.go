//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_server

import (
	"github.com/jfsmig/object-storage/pkg/blob-model"
	"os"
)

type Repo interface {
	Create(objid gunkan_blob_model.Id) (BlobBuilder, error)
	Open(blobid string) (BlobReader, error)
	Delete(blobid string) error
}

type BlobReader interface {
	Stream() *os.File
	Close()
}

type BlobBuilder interface {
	Stream() *os.File
	Commit() (string, error)
	Abort() error
}
