// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package wal

import (
	"github.com/go-kit/log"
	"github.com/polarsignals/wal/fs"
	"github.com/polarsignals/wal/metadb"
	"github.com/polarsignals/wal/segment"
	"github.com/polarsignals/wal/types"
	"github.com/prometheus/client_golang/prometheus"
)

// WithMetaStore is an option that allows a custom MetaStore to be provided to
// the WAL. If not used the default MetaStore is used.
func WithMetaStore(db types.MetaStore) walOpt {
	return func(w *WAL) {
		w.metaDB = db
	}
}

// WithSegmentFiler is an option that allows a custom SegmentFiler (and hence
// Segment Reader/Writer implementation) to be provided to the WAL. If not used
// the default SegmentFiler is used.
func WithSegmentFiler(sf types.SegmentFiler) walOpt {
	return func(w *WAL) {
		w.sf = sf
	}
}

// WithLogger is an option that allows a custom logger to be used.
func WithLogger(logger log.Logger) walOpt {
	return func(w *WAL) {
		w.logger = logger
	}
}

// WithSegmentSize is an option that allows a custom segmentSize to be set.
func WithSegmentSize(size int) walOpt {
	return func(w *WAL) {
		w.segmentSize = size
	}
}

// WithMetricsRegisterer is an option that allows specifying a custom prometheus
// metrics registerer.
func WithMetricsRegisterer(reg prometheus.Registerer) walOpt {
	return func(w *WAL) {
		w.reg = reg
	}
}

func (w *WAL) applyDefaultsAndValidate() error {
	// Defaults
	if w.logger == nil {
		w.logger = log.NewNopLogger()
	}
	if w.sf == nil {
		// These are not actually swappable via options right now but we override
		// them in tests. Only load the default implementations if they are not set.
		vfs := fs.New()
		w.sf = segment.NewFiler(w.dir, vfs)
	}
	if w.reg == nil {
		w.reg = prometheus.NewRegistry()
	}
	if w.metrics == nil {
		w.metrics = newWALMetrics(w.reg)
	}
	if w.metaDB == nil {
		w.metaDB = &metadb.BoltMetaDB{}
	}
	if w.segmentSize == 0 {
		w.segmentSize = DefaultSegmentSize
	}
	return nil
}
