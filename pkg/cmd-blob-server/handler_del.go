//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_server

func handleBlobDel(ctx *handlerContext, blobid string) {
	err := ctx.srv.repo.Delete(blobid)
	if err != nil {
		ctx.replyError(err)
	} else {
		ctx.success()
	}
}
