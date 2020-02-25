//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_server

import (
	"github.com/jfsmig/object-storage/pkg/blob-model"
	"io"
	"net/http"
)

func handleBlobPut(ctx *handlerContext, encoded string) {
	objid := gunkan_blob_model.Id{}
	if err := objid.Decode(string(encoded)); err != nil {
		ctx.replyCodeError(http.StatusBadRequest, err)
		return
	}

	f, err := ctx.srv.repo.Create(objid)
	if err != nil {
		ctx.replyError(err)
		return
	}

	var final string
	_, err = io.Copy(f.Stream(), ctx.Input())
	if err != nil {
		f.Abort()
		ctx.replyError(err)
	} else if final, err = f.Commit(); err != nil {
		ctx.replyError(err)
	} else {
		ctx.setHeader("Location", final)
		ctx.WriteHeader(http.StatusCreated)
	}
}
