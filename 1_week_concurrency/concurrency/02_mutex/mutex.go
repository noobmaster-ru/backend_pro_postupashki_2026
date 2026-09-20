package mutex

import (
	"runtime"
	"sync/atomic"

	"primitives/internal/futex"
)

const (
	free = iota
	held
	contended
)

// spinIterations — сколько раз покрутиться перед тем, как уйти спать в ядро.
const spinIterations = 30

type Mutex struct {
	state uint32
}

func (m *Mutex) Lock() {
	// Быстрый путь: замок свободен, забираем одним CAS без единого syscall.
	if atomic.CompareAndSwapUint32(&m.state, free, held) {
		return
	}

	// Немного покрутиться: если владелец отпустит замок через сотню
	// наносекунд, это дешевле, чем идти в ядро и обратно.
	for i := 0; i < spinIterations; i++ {
		if atomic.LoadUint32(&m.state) == free &&
			atomic.CompareAndSwapUint32(&m.state, free, held) {
			return
		}
		runtime.Gosched()
	}

	// Медленный путь. Swap(contended) делает два дела сразу:
	// помечает, что появился ждущий, и сообщает, что лежало до этого.
	// Если там было free — владелец только что отпустил, замок наш.
	// После пробуждения снова ставим contended, а не held: мы не знаем,
	// сколько ещё спящих осталось за нами, и следующий Unlock обязан
	// разбудить кого-то ещё.
	for atomic.SwapUint32(&m.state, contended) != free {
		futex.Wait(&m.state, contended)
	}
}

func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapUint32(&m.state, free, held)
}

func (m *Mutex) Unlock() {
	// Swap(free) атомарно освобождает и возвращает прошлое состояние.
	// От него зависит и паника, и решение, звать ли ядро.
	switch atomic.SwapUint32(&m.state, free) {
	case free:
		panic("unlock of unlocked mutex")
	case held:
		// Ждущих нет — syscall не нужен. Это самый частый случай.
	case contended:
		// Кто-то спит. Будим одного: остальные всё равно будут ждать
		// этот же замок, а WakeAll только создаст стадо.
		futex.Wake(&m.state)
	}
}
