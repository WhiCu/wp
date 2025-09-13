package main

import (
	ctx "context"
	"fmt"

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
	)
	workPool.MaxWorkersCount = 2
	workPool.Start()
	for i := 0; i < 10; i++ {
		if !workPool.Serve(&MyInput{Name: "test"}) {
			fmt.Println("worker pool is full")
		}
		fmt.Println(workPool.Status())
		// time.Sleep(time.Second)
	}
	workPool.Stop()
}

func myController(c context.Context[*MyInput]) error {
	fmt.Println("controller", c.GetValue().Name)
	return nil
}
