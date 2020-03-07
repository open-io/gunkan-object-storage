//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package gunkan

import (
	"context"
)

type IndexClient interface {
	Status(ctx context.Context) (IndexStats, error)

	Health(ctx context.Context) (string, error)

	Put(ctx context.Context, base, key, value string) error

	Get(ctx context.Context, base, key string) (string, error)

	Delete(ctx context.Context, base, key string) error

	List(ctx context.Context, base, marker string, max uint32) ([]IndexListItem, error)
}

type IndexListItem struct {
	Key     string
	Version uint64
}

type IndexStats struct {
	B_in     uint64 `json:"b_in"`
	B_out    uint64 `json:"b_out"`
	T_info   uint64 `json:"t_info"`
	T_health uint64 `json:"t_health"`
	T_status uint64 `json:"t_status"`
	T_put    uint64 `json:"t_put"`
	T_get    uint64 `json:"t_get"`
	T_delete uint64 `json:"t_delete"`
	T_list   uint64 `json:"t_list"`
	H_info   uint64 `json:"h_info"`
	H_health uint64 `json:"h_health"`
	H_status uint64 `json:"h_status"`
	H_put    uint64 `json:"h_put"`
	H_get    uint64 `json:"h_get"`
	H_delete uint64 `json:"h_delete"`
	H_list   uint64 `json:"h_list"`
	C_200    uint64 `json:"c_200"`
	C_400    uint64 `json:"c_400"`
	C_404    uint64 `json:"c_404"`
	C_409    uint64 `json:"c_409"`
	C_50X    uint64 `json:"c_50X"`
}
