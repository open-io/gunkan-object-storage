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
	"net/http"
)

func handleList() handler {
	return func(ctx *handlerContext) {
		switch ctx.Method() {
		case "GET", "HEAD":
			ctx.WriteHeader(http.StatusNotImplemented)
		default:
			ctx.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
