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

### Как это работает
```mermaid
flowchart LR
    subgraph Client
        S[Serve(T)]
    end
    subgraph Pool
        direction TB
        R[Стек готовых воркеров]
        C1[Создание воркера при нехватке]
        L[Ограничение \n MaxWorkersCount]
    end
    subgraph Workers
        direction TB
        W1[Worker 1]
        W2[Worker 2]
        Wn[Worker N]
    end

    S -->|задача T| R
    R -->|есть готовый| W1
    R -->|нет готового| C1
    C1 -->|если < Max| W2
    C1 -.->|если = Max| X((false из Serve))

    W1 -->|после обработки| R
    W2 -->|после обработки| R

    CTX[context.Context] -->|Start/Stop, cancel| Pool
    Pool --> EH[ErrHandler(error cause)]
```

