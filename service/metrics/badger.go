package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/collectors"

	_ "github.com/dgraph-io/badger/v2/y"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterBadgerMetrics() error {
	expvarCol := collectors.NewExpvarCollector(map[string]*prometheus.Desc{
		"badger_v2_disk_reads_total":     prometheus.NewDesc("archive_badger_disk_reads_total", "cumulative number of reads", nil, nil),
		"badger_v2_disk_writes_total":    prometheus.NewDesc("archive_badger_disk_writes_total", "cumulative number of writes", nil, nil),
		"badger_v2_read_bytes":           prometheus.NewDesc("archive_badger_read_bytes", "cumulative number of bytes read", nil, nil),
		"badger_v2_written_bytes":        prometheus.NewDesc("archive_badger_written_bytes", "cumulative number of bytes written", nil, nil),
		"badger_v2_gets_total":           prometheus.NewDesc("archive_badger_gets_total", "number of gets", nil, nil),
		"badger_v2_memtable_gets_total":  prometheus.NewDesc("archive_badger_memtable_gets_total", "number of memtable gets", nil, nil),
		"badger_v2_puts_total":           prometheus.NewDesc("archive_badger_puts_total", "number of puts", nil, nil),
		"badger_v2_blocked_puts_total":   prometheus.NewDesc("archive_badger_blocked_puts_total", "number of blocked puts", nil, nil),
		"badger_v2_pending_writes_total": prometheus.NewDesc("archive_badger_badger_pending_writes_total", "tracks the number of pending writes", []string{"path"}, nil),
		"badger_v2_lsm_bloom_hits_total": prometheus.NewDesc("archive_badger_lsm_bloom_hits_total", "number of LSM bloom hits", []string{"level"}, nil),
		"badger_v2_lsm_level_gets_total": prometheus.NewDesc("archive_badger_lsm_level_gets_total", "number of LSM gets", []string{"level"}, nil),
		"badger_v2_lsm_size_bytes":       prometheus.NewDesc("archive_badger_lsm_size_bytes", "size of the LSM in bytes", []string{"path"}, nil),
		"badger_v2_vlog_size_bytes":      prometheus.NewDesc("archive_badger_vlog_size_bytes", "size of the value log in bytes", []string{"path"}, nil),
	})

	err := prometheus.Register(expvarCol)
	if err != nil {
		return fmt.Errorf("failed to register badger metrics: %w", err)
	}
	return nil
}
