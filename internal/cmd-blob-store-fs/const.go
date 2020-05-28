// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd_blob_store_fs

import (
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"golang.org/x/sys/unix"
)

const (
	flagsCommon   int = unix.O_NOATIME | unix.O_NONBLOCK | unix.O_CLOEXEC
	flagsRO           = flagsCommon | unix.O_RDONLY
	flagsRW           = flagsCommon | unix.O_RDWR
	flagsCreate       = flagsRW | unix.O_EXCL | unix.O_CREAT
	flagsOpenDir      = flagsRO | unix.O_DIRECTORY | unix.O_PATH
	flagsOpenRead     = flagsRO
)

const (
	routeList  = "/v1/list"
	prefixData = "/v1/blob/"
	infoString = "gunkan/blob-store-" + gunkan.VersionString
)
