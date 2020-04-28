//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_data_gate

import (
	"github.com/jfsmig/object-storage/pkg/gunkan"
)

const (
	routeList  = "/v1/list"
	prefixData = "/v1/part/"
	infoString = "gunkan/data-gate-" + gunkan.VersionString
)

const (
	HeaderPrefixCommon     = "X-gk-"
	HeaderNameObjectPolicy = HeaderPrefixCommon + "obj-policy"
)
