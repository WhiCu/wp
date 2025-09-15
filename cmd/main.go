package main

import (
	ctx "context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/WhiCu/wp"
)

type MyInput struct {
	Name string `json:"name" validate:"required"`
}

type MyOutput struct {
	Message string `json:"message"`
}

func main() {
	var i atomic.Uint32

	workPool := wp.New(
		myController(&i),
		wp.WithMaxWorkersCount[*MyInput](10),
		wp.WithMaxIdleWorkerDuration[*MyInput](time.Second),
		wp.WithErrHandler[*MyInput]( // Обработчик ошибок
			func(err wp.Context[error]) {
				fmt.Println("error", err.GetValue())
			},
		),
	)

	workPool.Start(ctx.Background())

	for i := 0; i < 10; i++ {
		if !workPool.Serve(&MyInput{Name: fmt.Sprint("hello", i)}) {
			fmt.Println("worker pool is full")
		}
		fmt.Println(workPool.Status())
	}

	fmt.Println(workPool.Status())

	workPool.Stop()
	workPool.Stop()

	fmt.Println("i", i.Load())
}

func myController(i *atomic.Uint32) wp.HandlerFunc[*MyInput] {
	return func(c wp.Context[*MyInput]) {
		i.Add(1)
		fmt.Println("controller", c.GetValue().Name)

		if i.Load() == 5 {
			// panic("panic")
		}
	}
	// time.Sleep(10 * time.Second)
}
