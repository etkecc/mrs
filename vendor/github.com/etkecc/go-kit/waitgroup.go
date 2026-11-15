package kit

import "sync"

// WaitGroup wraps sync.WaitGroup with Do/Wait; Do is non-blocking. Zero value unusable, use NewWaitGroup.
type WaitGroup struct {
	wg *sync.WaitGroup
}

// NewWaitGroup returns a WaitGroup ready to use.
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{wg: &sync.WaitGroup{}}
}

// Do launches each f in its own goroutine, returns immediately, wraps Add/Done; nil func panics; callable repeatedly.
func (w *WaitGroup) Do(f ...func()) {
	w.wg.Add(len(f))
	for _, fn := range f {
		go func() {
			defer w.wg.Done()
			fn()
		}()
	}
}

// Get returns the underlying *sync.WaitGroup for stdlib-shaped code or ops Do doesn't wrap.
func (w *WaitGroup) Get() *sync.WaitGroup {
	return w.wg
}

// Wait blocks until all goroutines launched via Do have returned; returns immediately if Do was never called.
func (w *WaitGroup) Wait() {
	w.wg.Wait()
}
