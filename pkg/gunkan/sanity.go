// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

func ValidateBucketName(n string) bool {
	return len(n) > 0 && len(n) < 1024
}

func ValidateContentName(n string) bool {
	return len(n) > 0 && len(n) < 1024
}
