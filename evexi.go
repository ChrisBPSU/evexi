package evexi

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Errors
var (
	ErrExportFuncNotSet       = errors.New("export function was not set")
	ErrNegativeOrZeroInterval = errors.New("the export interval was negative or zero")
)

// Evexi should be initialized by the New func
type Evexi struct {
	buffer []byte

	// How to export (eg. Write to disk, cloud, etc)
	// TODO: potentially consider a no-copy export option for larger buffers, not sure how to implement it without mux locking
	export func([]byte)
	// When set, the buffer will always remain this size, and when exceeded a new buffer of this size will be created and the last GC'd
	bufferMaxSize int

	// Detail used to keep the buffer size consistent over time
	// Only used when bufMaxSize is not set
	totalBufSize int
	bufCount     int

	mux sync.Mutex
}

// New exports function must not be nil
func New(export func([]byte), bufferMaxSize int) (*Evexi, error) {
	if export == nil {
		return nil, ErrExportFuncNotSet
	}

	initialBufSize := 0

	// BufferMaxSize
	if bufferMaxSize > 0 {
		initialBufSize = bufferMaxSize
	}

	e := &Evexi{
		buffer: make([]byte, 0, initialBufSize),

		export:        export,
		bufferMaxSize: bufferMaxSize,

		mux: sync.Mutex{},
	}

	return e, nil
}

// Export calls the underlying export func with mux locking and the option to reset the buffer
func (e *Evexi) Export(reset bool) {
	e.mux.Lock()
	defer e.mux.Unlock()

	data := e.bytes()

	if reset {
		e.reset()
	}

	go e.export(data)
}

// IntervalExport must recieve a positive interval
// Returns a cancellable goroutine that exports the buffer at the provided interval
func (e *Evexi) IntervalExport(exportInterval time.Duration) (func(), error) {
	if exportInterval <= 0 {
		return nil, ErrNegativeOrZeroInterval
	}

	// Start goroutine to export the buffer at a regular interval
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		t := time.NewTicker(exportInterval)

		for {
			select {
			case <-ctx.Done():
				// Stop the ticker and exit
				t.Stop()
				return
			case <-t.C:
				// Lock the buffer
				// Copy the data
				// Reset the buffer
				e.mux.Lock()
				data := e.bytes()
				e.reset()
				e.mux.Unlock()

				// Export the data
				go e.export(data)
			}
		}
	}()

	return cancel, nil
}

func (e *Evexi) Write(data []byte) (n int, err error) {
	e.mux.Lock()
	defer e.mux.Unlock()

	// Check bufferMaxSize is set
	if e.bufferMaxSize > 0 {
		// Check if new data will exceed bufferMaxSize
		if len(e.buffer)+len(data) > e.bufferMaxSize {
			// Copy the data
			// Reset the buffer
			data := e.bytes()
			e.reset()

			// Export the data
			go e.export(data)
		}
	}

	// Add the data to the buffer
	e.buffer = append(e.buffer, data...)

	return len(data), nil
}

// Bytes makes a copy of the data
func (e *Evexi) Bytes() []byte {
	e.mux.Lock()
	defer e.mux.Unlock()

	return e.bytes()
}
func (e *Evexi) bytes() []byte {
	bufCopy := make([]byte, len(e.buffer))
	copy(bufCopy[0:], e.buffer)

	return bufCopy
}

// Reset resets the buffer
func (e *Evexi) Reset() {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.reset()
}

// bufferMaxSize keeps the buffer size consistent
// avgBufSize stops the buffer from growing too much and wasting too much memory (very rudimentary implementation)
func (e *Evexi) reset() {
	// check bufferMaxSize is set
	if e.bufferMaxSize > 0 {
		if len(e.buffer) >= e.bufferMaxSize {
			// the bufferMaxSize was exceeded, creating a new buffer
			e.buffer = make([]byte, 0, e.bufferMaxSize)
		} else {
			// reuse current buffer
			e.buffer = e.buffer[:0]
		}

		return
	}

	// TODO: consider using cap instead of len
	// bufferMaxSize is not set, using avgBufSize
	e.bufCount++
	e.totalBufSize += len(e.buffer)

	avgBufSize := e.totalBufSize / e.bufCount

	// Twice the size of the average buffer is used so that when it goes over there won't be a need for an allocation
	if len(e.buffer) <= avgBufSize*2 {
		// Reuse buffer
		e.buffer = e.buffer[:0]
	} else {
		// Create a new buffer if the current buffer exceeds twice the size of the avgBufferSize
		e.buffer = make([]byte, avgBufSize)
	}
}
