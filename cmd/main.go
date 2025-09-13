package main

import (
	ctx "context"
	"fmt"
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
	workPool := wp.New(
		myController,
		ctx.Background(),
		wp.WithMaxWorkersCount[*MyInput](10),
		wp.WithMaxIdleWorkerDuration[*MyInput](time.Second),
	)
	workPool.Start()
	for i := 0; i < 10; i++ {
		// fmt.Println("start", i)
		if !workPool.Serve(&MyInput{Name: fmt.Sprint("hello", i)}) {
			fmt.Println("worker pool is full")
		}
		fmt.Println(workPool.Status())
		// time.Sleep(time.Second)
	}
	// fmt.Println(workPool.Status())
	// time.Sleep(3 * time.Second)
	// fmt.Println(workPool.Status())
	// for i := 0; i < 10; i++ {
	// 	// fmt.Println("start", i)
	// 	if !workPool.Serve(&MyInput{Name: fmt.Sprint("hello", i)}) {
	// 		fmt.Println("worker pool is full")
	// 	}
	// 	fmt.Println(workPool.Status())
	// 	// time.Sleep(time.Second)
	// }
	// fmt.Println(workPool.Status())
	time.Sleep(5 * time.Second)
	fmt.Println(workPool.Status())
	workPool.Stop()
}

func myController(c context.Context[*MyInput]) error {
	// time.Sleep(10 * time.Second)
	fmt.Println("controller", c.GetValue().Name)
	return nil
}
