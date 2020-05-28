// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
