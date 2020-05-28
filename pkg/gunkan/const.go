// Copyright (C) 2019-2020 OpenIO SAS
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gunkan

const (
	VersionMajor  = "0"
	VersionMinor  = "1"
	VersionString = VersionMajor + "." + VersionMinor
)

const (
	ConsulSrvIndexGate  = "gkindex-gate"
	ConsulSrvIndexStore = "gkindex-store"
	ConsulSrvDataGate   = "gkdata-gate"
	ConsulSrvBlobStore  = "gkblob-store"
)

const (
	// Returns a simple string containing the type of the service.
	RouteInfo = "/info"

	// Provides the feedback expected by Consul.io
	RouteHealth = "/health"

	// Provides metrics using the standard of a Prometheus Exporter
	// Thus also collectable with InfluxDB
	RouteMetrics = "/metrics"
)

const (
	ListHardMax = 10000
)
