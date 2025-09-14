package main

import (
	ctx "context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/WhiCu/wp"
	"github.com/WhiCu/wp/context"
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
		wp.WithErrHandler[*MyInput](func(err error) {
			fmt.Println("ErrHandler", err)
		}),
	)
	workPool.Start(ctx.Background())
	for i := 0; i < 1000; i++ {
		// fmt.Println("start", i)
		if !workPool.Serve(&MyInput{Name: fmt.Sprint("hello", i)}) {
			fmt.Println("worker pool is full")
		}
		fmt.Println(workPool.Status())
		// time.Sleep(time.Second)
	}
	time.Sleep(5 * time.Second)
	fmt.Println(workPool.Status())
	// fmt.Println("stop 1")
	workPool.Stop()
	fmt.Println("i", i.Load())

	// workPool.Start(ctx.Background())
	// for i := 0; i < 10; i++ {
	// 	// fmt.Println("start", i)
	// 	if !workPool.Serve(&MyInput{Name: fmt.Sprint("hello", i)}) {
	// 		fmt.Println("worker pool is full")
	// 	}
	// 	fmt.Println(workPool.Status())
	// 	// time.Sleep(time.Second)
	// }
	// // time.Sleep(5 * time.Second)
	// fmt.Println(workPool.Status())
	// fmt.Println("stop 2")
	// workPool.Stop()
}

func myController(i *atomic.Uint32) wp.HandlerFunc[*MyInput] {
	return func(c context.Context[*MyInput]) error {
		i.Add(1)
		fmt.Println("controller", c.GetValue().Name)
		return nil
	}
	// time.Sleep(10 * time.Second)

}
