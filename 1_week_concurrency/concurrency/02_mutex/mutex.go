package mutex

import (
	"sync/atomic"
)

const (
	free = iota
	held
	contended
)

type Mutex struct {
	state uint32
}

func (m *Mutex) Lock() {
	mutexState := atomic.LoadUint32(&m.state)
	if mutexState == free {
		if atomic.CompareAndSwapUint32(&m.state, free, held) {
			return
		}
	}
	
	for {
		if mutexState == contended || atomic.CompareAndSwapUint32(&m.state, held, contended) {
			// Wait for the lock to be released
			for atomic.LoadUint32(&m.state) != free {
				// Busy wait
			}
		}
		if atomic.CompareAndSwapUint32(&m.state, free, held) {
			return
		}
		mutexState = atomic.LoadUint32(&m.state)
	}
}

func (m *Mutex) TryLock() bool {
	mutexState := atomic.LoadUint32(&m.state)
	if mutexState == free {
		return atomic.CompareAndSwapUint32(&m.state, free, held)
	}
	return false
}

func (m *Mutex) Unlock() {
	mutexState := atomic.LoadUint32(&m.state)
	if mutexState == held {
		if !atomic.CompareAndSwapUint32(&m.state, held, free) {
			panic("unlock of unlocked mutex")
		}
	} else if mutexState == contended {
		if !atomic.CompareAndSwapUint32(&m.state, contended, free) {
			panic("unlock of unlocked mutex")
		}
	} else {
		panic("unlock of unlocked mutex")
	}	
}
