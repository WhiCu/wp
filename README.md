## WP — worker pool на Go

Пул воркеров с настраиваемыми границами параллелизма. Позволяет обрабатывать входящие задачи типобезопасным обработчиком.

### Возможности
- **Min/Max воркеры**: нижняя и верхняя границы количества воркеров
- **Idle-очистка**: завершение неиспользуемых воркеров по таймауту
- **Контекст**: запуск с внешним `context.Context`, корректная остановка
- **Обработчик ошибок**: хук для причин остановки воркеров

### Установка
```bash
go get github.com/WhiCu/wp@latest
```

### Пример (см. `examples/main.go`)
```go


var (
	out        = bufio.NewWriter(os.Stdout)
	numWorkers = flag.Uint("workers", 1, "base number of workers")
	maxWorkers = flag.Uint("max-workers", 10, "maximum additional workers that can be created on top of base workers (total: workers + max-workers)")
)

type MyInput struct {
	Name string
}

func HandlerFunc(c wp.Context[*MyInput]) {
	fmt.Fprintf(out, "handle: %s\n", c.GetValue().Name)
}

func main() {
	flag.Parse()
	var i atomic.Uint32

	pool := wp.New(
		HandlerFunc,
		wp.WithMinWorkersCount[*MyInput](uint32(*numWorkers)),
		wp.WithMaxWorkersCount[*MyInput](uint32(*numWorkers+*maxWorkers)),
		wp.WithMaxIdleWorkerDuration[*MyInput](time.Second),
		wp.WithErrHandler[*MyInput](func(err wp.Context[error]) {
			fmt.Println("worker error:", err.GetValue())
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool.Start(ctx)

	for i := 0; i < 10; i++ {
		ok := pool.Serve(&MyInput{Name: fmt.Sprint("job-", i)})
		if !ok {
			fmt.Println("pool is full")
		}
		fmt.Println(pool.Status())
	}

	time.Sleep(2 * time.Second)
	fmt.Println("Status:", pool.Status())
	pool.Stop()
	fmt.Println("Status:", pool.Status())
	fmt.Println("i:", i.Load())
}


```

#### ВЫВОД

```bash

-WorkersCount: 1	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 1	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 2	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 3	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 4	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 5	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 6	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 7	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 8	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
-WorkersCount: 9	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
worker error: idle worker
worker error: idle worker
worker error: idle worker
worker error: idle worker
worker error: idle worker
worker error: idle worker
worker error: idle worker
worker error: idle worker
Status: -WorkersCount: 1	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
worker error: SIGTERM
Status: -WorkersCount: 0	↑MaxWorkersCount: 11	↓MinWorkersCount: 1
i: 0

```

### Публичный интерфейс
- **`wp.New(handler, ...options)`**: создать пул с обработчиком типа `func(c wp.Context[T])`
- **`Start(ctx)`**: запустить пул и систему воркеров
- **`Serve(t T) bool`**: передать задачу; `false`, если нет готовых воркеров и достигнут максимум
- **`Stop()`**: остановить пул, дождаться завершения воркеров
- **`Status() string`**: краткая сводка по состоянию
- **Опции**:
  - `WithMinWorkersCount[T](n uint32)`
  - `WithMaxWorkersCount[T](n uint32)`
  - `WithMaxIdleWorkerDuration[T](d time.Duration)`
  - `WithErrHandler[T](h ErrHandler)` где `ErrHandler = func(c wp.Context[error])`
