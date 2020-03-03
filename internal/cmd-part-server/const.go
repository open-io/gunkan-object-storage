//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_part_server

import (
	"github.com/jfsmig/object-storage/pkg/gunkan"
)

const (
	routeInfo   = "/v1/info"
	routeHealth = "/v1/health"
	routeStatus = "/v1/status"
	routeList   = "/v1/list"
	prefixBlob  = "/v1/part/"
	infoString  = "gunkan/blob-part-" + gunkan.VersionString
)
