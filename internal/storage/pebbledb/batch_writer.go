package pebbledb

import (
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
)

type BatchWriterConfig struct {
	MaxBatchSize      int // Flush after this many ops (default: 1000)
	ChannelBufferSize int
}

func DefaultBatchWriterConfig() BatchWriterConfig {
	return BatchWriterConfig{
		MaxBatchSize:      1000,
		ChannelBufferSize: 1000000, // Large buffer for bursts
	}
}

type writeOp struct {
	key    []byte
	value  []byte
	delete bool
	merge  bool
}

type BatchWriter struct {
	db      *pebble.DB
	config  BatchWriterConfig
	opCh    chan writeOp
	stopCh  chan struct{}
	doneCh  chan struct{}
	stopped atomic.Bool
}

func NewBatchWriter(db *pebble.DB, config BatchWriterConfig) *BatchWriter {
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 1000
	}
	if config.ChannelBufferSize == 0 {
		config.ChannelBufferSize = 100000
	}

	bw := &BatchWriter{
		db:     db,
		config: config,
		opCh:   make(chan writeOp, config.ChannelBufferSize),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	go bw.flusher()

	return bw
}

// Set queues a Set operation (lock-free)
func (bw *BatchWriter) Set(key, value []byte) {
	if bw.stopped.Load() {
		return
	}
	bw.opCh <- writeOp{key: key, value: value}
}

func (bw *BatchWriter) Delete(key []byte) {
	if bw.stopped.Load() {
		return
	}
	bw.opCh <- writeOp{key: key, delete: true}
}

func (bw *BatchWriter) Merge(key, value []byte) {
	if bw.stopped.Load() {
		return
	}
	bw.opCh <- writeOp{key: key, value: value, merge: true}
}

func (bw *BatchWriter) Close() error {
	if bw.stopped.Swap(true) {
		return nil // Already stopped
	}
	close(bw.stopCh)
	<-bw.doneCh // Wait for flusher to finish
	return nil
}

func (bw *BatchWriter) flusher() {
	defer close(bw.doneCh)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	batch := bw.db.NewBatch()
	opCount := 0

	flush := func() {
		if opCount == 0 {
			return
		}
		if err := batch.Commit(pebble.Sync); err != nil {
			// Log error but continue - we don't want to crash the server
			// In production, you might want better error handling
		}
		batch.Close()
		batch = bw.db.NewBatch()
		opCount = 0
	}

	for {
		select {
		case op, ok := <-bw.opCh:
			if !ok {
				// Channel closed, flush remaining
				flush()
				batch.Close()
				return
			}

			// Add operation to batch
			switch {
			case op.delete:
				batch.Delete(op.key, nil)
			case op.merge:
				batch.Merge(op.key, op.value, nil)
			default:
				batch.Set(op.key, op.value, nil)
			}
			opCount++

			// Flush when batch is full (1000 ops)
			if opCount >= bw.config.MaxBatchSize {
				flush()
			}

		case <-ticker.C:
			// Time-based flush every 1 second
			flush()

		case <-bw.stopCh:
			// Drain remaining operations from channel
			for {
				select {
				case op, ok := <-bw.opCh:
					if !ok {
						flush()
						batch.Close()
						return
					}
					switch {
					case op.delete:
						batch.Delete(op.key, nil)
					case op.merge:
						batch.Merge(op.key, op.value, nil)
					default:
						batch.Set(op.key, op.value, nil)
					}
					opCount++
				default:
					// Channel drained
					flush()
					batch.Close()
					return
				}
			}
		}
	}
}
