// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package metrics

import (
	"fmt"

	_ "github.com/dgraph-io/badger/v2/y"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterBadgerMetrics() error {
	expvarCol := prometheus.NewExpvarCollector(map[string]*prometheus.Desc{
		"badger_v2_disk_reads_total":     prometheus.NewDesc("badger_disk_reads_total", "cumulative number of reads", nil, nil),
		"badger_v2_disk_writes_total":    prometheus.NewDesc("badger_disk_writes_total", "cumulative number of writes", nil, nil),
		"badger_v2_read_bytes":           prometheus.NewDesc("badger_read_bytes", "cumulative number of bytes read", nil, nil),
		"badger_v2_written_bytes":        prometheus.NewDesc("badger_written_bytes", "cumulative number of bytes written", nil, nil),
		"badger_v2_gets_total":           prometheus.NewDesc("badger_gets_total", "number of gets", nil, nil),
		"badger_v2_memtable_gets_total":  prometheus.NewDesc("badger_memtable_gets_total", "number of memtable gets", nil, nil),
		"badger_v2_puts_total":           prometheus.NewDesc("badger_puts_total", "number of puts", nil, nil),
		"badger_v2_blocked_puts_total":   prometheus.NewDesc("badger_blocked_puts_total", "number of blocked puts", nil, nil),
		"badger_v2_pending_writes_total": prometheus.NewDesc("badger_badger_pending_writes_total", "tracks the number of pending writes", []string{"path"}, nil),
		"badger_v2_lsm_bloom_hits_total": prometheus.NewDesc("badger_lsm_bloom_hits_total", "number of LSM bloom hits", []string{"level"}, nil),
		"badger_v2_lsm_level_gets_total": prometheus.NewDesc("badger_lsm_level_gets_total", "number of LSM gets", []string{"level"}, nil),
		"badger_v2_lsm_size_bytes":       prometheus.NewDesc("badger_lsm_size_bytes", "size of the LSM in bytes", []string{"path"}, nil),
		"badger_v2_vlog_size_bytes":      prometheus.NewDesc("badger_vlog_size_bytes", "size of the value log in bytes", []string{"path"}, nil),
	})

	err := prometheus.Register(expvarCol)
	if err != nil {
		return fmt.Errorf("failed to register badger metrics: %w", err)
	}
	return nil
}
