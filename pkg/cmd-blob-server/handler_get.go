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
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"net/http"
)

func handleBlobGet(ctx *handlerContext, blobid string) {
	var st unix.Stat_t
	var f BlobReader
	var err error

	f, err = ctx.srv.repo.Open(blobid)
	if err != nil {
		ctx.replyError(err)
		return
	} else {
		defer f.Close()
	}

	err = unix.Fstat(int(f.Stream().Fd()), &st)
	if err == nil {
		if st.Size == 0 {
			ctx.setHeader("Content-Length", "0")
			ctx.WriteHeader(http.StatusNoContent)
		} else {
			ctx.setHeader("Content-Length", fmt.Sprintf("%d", st.Size))
			ctx.WriteHeader(http.StatusOK)
		}
		ctx.setHeader("Content-Type", "octet/stream")
		_, err = io.Copy(ctx.Output(), &io.LimitedReader{R: f.Stream(), N: st.Size})
	}
	if err != nil {
		ctx.replyError(err)
	}
}
