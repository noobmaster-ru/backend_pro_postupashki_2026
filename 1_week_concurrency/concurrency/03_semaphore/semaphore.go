package semaphore

import (
	"sync/atomic"

	"primitives/internal/futex"
)

// Semaphore — счётный семафор Дейкстры на одном слове памяти.
// permits хранит число свободных разрешений. Владельца у семафора нет:
// Release может позвать любая горутина, не обязательно та, что делала Acquire.
type Semaphore struct {
	permits uint32
}

func New(n int) *Semaphore {
	if n < 0 {
		panic("semaphore: отрицательное число разрешений")
	}
	return &Semaphore{permits: uint32(n)}
}

// Acquire забирает одно разрешение, блокируясь, пока их нет.
func (s *Semaphore) Acquire() {
	for {
		p := atomic.LoadUint32(&s.permits)
		if p == 0 {
			// Уснуть, но только если разрешений всё ещё ноль. Если между
			// Load и Wait кто-то успел сделать Release, Wait вернётся сразу.
			futex.Wait(&s.permits, 0)
			continue
		}
		// Разрешение есть, но между Load и уменьшением его могли
		// перехватить — поэтому не Add(-1), а CAS от увиденного значения.
		if atomic.CompareAndSwapUint32(&s.permits, p, p-1) {
			return
		}
	}
}

// TryAcquire забирает разрешение без ожидания. Цикл только из-за
// проигранного CAS: если разрешений ноль, сразу возвращаем false.
func (s *Semaphore) TryAcquire() bool {
	for {
		p := atomic.LoadUint32(&s.permits)
		if p == 0 {
			return false
		}
		if atomic.CompareAndSwapUint32(&s.permits, p, p-1) {
			return true
		}
	}
}

// Release возвращает разрешение и будит одного ждущего.
// Ждущих мы не считаем, поэтому будим всегда: если никто не спит,
// Wake просто ничего не сделает.
func (s *Semaphore) Release() {
	atomic.AddUint32(&s.permits, 1)
	futex.Wake(&s.permits)
}

func (s *Semaphore) Available() int {
	return int(atomic.LoadUint32(&s.permits))
}
