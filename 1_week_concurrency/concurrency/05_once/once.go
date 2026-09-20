package once

import (
	"sync/atomic"

	"primitives/internal/futex"
)

const (
	notStarted = iota // ещё никто не начинал
	running           // победитель гонки выполняет f, остальные ждут
	done              // f отработала (или упала), больше не запускаем
)

type Once struct {
	state uint32
}

func (o *Once) Do(f func()) {
	// Быстрый путь: работа уже сделана. Именно сюда попадают все вызовы
	// после первого, поэтому здесь одно атомарное чтение и ничего больше.
	if atomic.LoadUint32(&o.state) == done {
		return
	}

	// Гонка за право выполнить f. CAS выигрывает ровно одна горутина.
	if atomic.CompareAndSwapUint32(&o.state, notStarted, running) {
		// Переход в done и побудка ждущих в defer: если f паникует,
		// вызов всё равно считается состоявшимся, ждущие не зависнут,
		// а паника уйдёт дальше к вызывающему.
		defer func() {
			atomic.StoreUint32(&o.state, done)
			futex.WakeAll(&o.state)
		}()
		f()
		return
	}

	// Проиграли гонку: кто-то выполняет f прямо сейчас. Ждём, пока
	// он закончит, иначе вернёмся к наполовину построенному состоянию.
	// Спим на значении running: если победитель уже успел поставить done,
	// Wait вернётся сразу.
	for atomic.LoadUint32(&o.state) != done {
		futex.Wait(&o.state, running)
	}
}

func (o *Once) Done() bool {
	return atomic.LoadUint32(&o.state) == done
}
