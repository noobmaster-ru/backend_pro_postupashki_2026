package waitgroup

import (
	"sync/atomic"

	"primitives/internal/futex"
)

// WaitGroup — счётчик незавершённой работы на одном слове памяти.
// Add увеличивает, Done уменьшает, Wait спит, пока счётчик не станет нулём.
type WaitGroup struct {
	count uint32
}

func (wg *WaitGroup) Add(delta int) {
	// Отрицательная delta в uint32 заворачивается по модулю 2^32,
	// а сложение по модулю даёт тот же результат, что и вычитание.
	// Читаем результат как int32, чтобы увидеть уход в минус.
	n := int32(atomic.AddUint32(&wg.count, uint32(delta)))
	if n < 0 {
		panic("waitgroup: отрицательный счётчик")
	}
	if n == 0 {
		// Работа закончилась. Будим всех: каждому ждущему нужен
		// именно этот момент, делить между ними нечего.
		futex.WakeAll(&wg.count)
	}
}

func (wg *WaitGroup) Done() {
	wg.Add(-1)
}

func (wg *WaitGroup) Wait() {
	for {
		c := atomic.LoadUint32(&wg.count)
		if c == 0 {
			return
		}
		// Спим на том значении, что только что прочитали. Если между
		// Load и Wait счётчик уже изменился (в том числе дошёл до нуля
		// и WakeAll отгремел без нас), Wait вернётся сразу, и цикл
		// перечитает счётчик.
		futex.Wait(&wg.count, c)
	}
}
