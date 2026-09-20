package barrier

import (
	"sync/atomic"

	"primitives/internal/futex"
)

// Barrier — точка сбора на N участников, переиспользуемая по раундам.
type Barrier struct {
	need    uint32 // сколько участников нужно собрать
	arrived uint32 // сколько уже пришло в текущем раунде
	round   uint32 // номер раунда; ждущие спят, пока он не сменится
}

func New(n int) *Barrier {
	if n <= 0 {
		panic("barrier: число участников должно быть положительным")
	}
	return &Barrier{need: uint32(n)}
}

func (b *Barrier) Wait() {
	// Запоминаем раунд ДО того, как отметиться. Иначе последний участник
	// может успеть сменить раунд между нашим Add и Load, и мы уснём,
	// дожидаясь смены уже нового номера.
	r := atomic.LoadUint32(&b.round)

	if atomic.AddUint32(&b.arrived, 1) == b.need {
		// Мы последние. Сначала обнуляем счётчик прихода, потом открываем
		// раунд: к моменту, когда кто-то проснётся и войдёт в барьер снова,
		// счётчик уже должен быть готов к следующему кругу.
		atomic.StoreUint32(&b.arrived, 0)
		atomic.AddUint32(&b.round, 1)
		futex.WakeAll(&b.round)
		return
	}

	// Спим, пока номер раунда равен запомненному. Смена номера — сигнал,
	// который нельзя проспать (Wait проверяет значение атомарно) и нельзя
	// перепутать со следующим раундом (у него будет другой номер).
	for atomic.LoadUint32(&b.round) == r {
		futex.Wait(&b.round, r)
	}
}
