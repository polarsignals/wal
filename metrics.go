// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package wal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	BytesWritten          prometheus.Counter
	EntriesWritten        prometheus.Counter
	Appends               prometheus.Counter
	EntryBytesRead        prometheus.Counter
	EntriesRead           prometheus.Counter
	SegmentRotations      prometheus.Counter
	EntriesTruncated      *prometheus.CounterVec
	Truncations           *prometheus.CounterVec
	LastSegmentAgeSeconds prometheus.Gauge
}

func newWALMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		BytesWritten: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "entry_bytes_written",
			Help: "entry_bytes_written counts the bytes of log entry after encoding." +
				" Actual bytes written to disk might be slightly higher as it" +
				" includes headers and index entries.",
		}),
		EntriesWritten: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "entries_written",
			Help: "entries_written counts the number of entries written.",
		}),
		Appends: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "appends",
			Help: "appends counts the number of calls to StoreLog(s) i.e." +
				" number of batches of entries appended.",
		}),
		EntryBytesRead: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "entry_bytes_read",
			Help: "entry_bytes_read counts the bytes of log entry read from" +
				" segments before decoding. actual bytes read from disk might be higher" +
				" as it includes headers and index entries and possible secondary reads" +
				" for large entries that don't fit in buffers.",
		}),
		EntriesRead: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "entries_read",
			Help: "entries_read counts the number of calls to get_log.",
		}),
		SegmentRotations: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "segment_rotations",
			Help: "segment_rotations counts how many times we move to a new segment file.",
		}),
		EntriesTruncated: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "entries_truncated_total",
				Help: "entries_truncated counts how many log entries have been truncated" +
					" from the front or back.",
			},
			[]string{"type"},
		),
		Truncations: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "truncations_total",
				Help: "truncations is the number of truncate calls categorized by whether" +
					" the call was successful or not.",
			},
			[]string{"type", "success"},
		),
		LastSegmentAgeSeconds: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "last_segment_age_seconds",
			Help: "last_segment_age_seconds is a gauge that is set each time we" +
				" rotate a segment and describes the number of seconds between when" +
				" that segment file was first created and when it was sealed. this" +
				" gives a rough estimate how quickly writes are filling the disk.",
		}),
	}
}
