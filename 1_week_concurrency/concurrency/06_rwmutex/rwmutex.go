package rwmutex

import (
	"sync/atomic"

	"primitives/internal/futex"
)

// Раскладка state: старший бит — «работает писатель», младшие 31 бит —
// число активных читателей. «Свободно» — это ровно ноль.
const writer = 1 << 31

type RWMutex struct {
	state uint32
}

func (rw *RWMutex) RLock() {
	for {
		s := atomic.LoadUint32(&rw.state)
		if s&writer != 0 {
			// Внутри писатель. Спим на увиденном значении: если он уже
			// вышел и слово изменилось, Wait вернётся сразу.
			futex.Wait(&rw.state, s)
			continue
		}
		// Писателя нет — становимся ещё одним читателем. CAS, а не Add:
		// между Load и увеличением писатель мог успеть зайти.
		if atomic.CompareAndSwapUint32(&rw.state, s, s+1) {
			return
		}
	}
}

func (rw *RWMutex) RUnlock() {
	for {
		s := atomic.LoadUint32(&rw.state)
		if s&writer != 0 || s == 0 {
			panic("rwmutex: RUnlock без RLock")
		}
		if atomic.CompareAndSwapUint32(&rw.state, s, s-1) {
			if s-1 == 0 {
				// Ушёл последний читатель — теперь может зайти писатель.
				futex.WakeAll(&rw.state)
			}
			return
		}
	}
}

func (rw *RWMutex) Lock() {
	for {
		// Писатель заходит только в полностью свободный замок.
		if atomic.CompareAndSwapUint32(&rw.state, 0, writer) {
			return
		}
		s := atomic.LoadUint32(&rw.state)
		if s != 0 {
			futex.Wait(&rw.state, s)
		}
	}
}

func (rw *RWMutex) Unlock() {
	if !atomic.CompareAndSwapUint32(&rw.state, writer, 0) {
		panic("rwmutex: Unlock без Lock")
	}
	// Ждать могли и читатели, и писатели. Читателям можно всем сразу,
	// поэтому будим всех; проигравшие писатели снова уснут.
	futex.WakeAll(&rw.state)
}
