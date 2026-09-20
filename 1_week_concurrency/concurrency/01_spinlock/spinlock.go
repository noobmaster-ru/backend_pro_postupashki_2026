package spinlock

import (
	"runtime"
	"sync/atomic"
)

type Spinlock struct {
	locked atomic.Bool
}

func (s *Spinlock) Lock() {
	for {
		if s.locked.CompareAndSwap(false, true) {
			return
		}else{
			runtime.Gosched()
		}
	}
}

func (s *Spinlock) TryLock() bool {
	return s.locked.CompareAndSwap(false, true)
}

func (s *Spinlock) Unlock() {
	if s.locked.Swap(false){
		return 
	}else{
		panic("unlock of unlocked spinlock")
	}
}

type TTAS struct {
	locked atomic.Bool
}

func (s *TTAS) Lock() {
	for {
		for s.locked.Load() {
			runtime.Gosched()
		}	
		if s.locked.CompareAndSwap(false, true) {
			return 
		}
	}	
}

func (s *TTAS) TryLock() bool {
	if s.locked.Load() {
		return false
	}else{
		return s.locked.CompareAndSwap(false, true)
	}
}

func (s *TTAS) Unlock() {
	if s.locked.Swap(false){
		return 
	} else{
		panic("unlock of unlocked spinlock")
	}
}
