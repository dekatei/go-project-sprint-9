package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var x int64 = 1
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ch <- x
			fn(x)
			x++

		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out)
	for x := range in {
		out <- x
		time.Sleep(time.Millisecond)
	}

}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum atomic.Int64   // сумма сгенерированных чисел
	var inputCount atomic.Int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum.Add(i)
		inputCount.Add(1)
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs
	for i := range outs {
		wg.Add(1)
		go func(in <-chan int64, i int64) { // вообще не поняла, что произошло. но вроде работает. Просто написала код по ТЗ аооаоааоаоао
			defer wg.Done()
			for x := range in { //это тоже не сработало
				chOut <- x
				amounts[i]++
			}

		}(outs[i], int64(i))

	}
	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала

	for x := range chOut {

		//atomic.AddInt64(&sum, x)
		//atomic.AddInt64(&count, 1) //не нужно, тк нет гонки
		sum += x
		count += 1
	}

	fmt.Println("Количество чисел", inputCount.Load(), count)
	fmt.Println("Сумма чисел", inputSum.Load(), sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum.Load() != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum.Load(), sum)
	}
	if inputCount.Load() != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount.Load(), count)
	}
	for _, v := range amounts {
		inputCount.Add(-v)
	}
	if inputCount.Load() != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}

}
