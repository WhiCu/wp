package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/WhiCu/wp"
)

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
}
