package pool

import (
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPool(t *testing.T) {
	Convey("Given a pool of ints", t, func() {
		p := New(func() int { return 42 })

		Convey("When getting a value", func() {
			v := p.Get()

			Convey("Then it should return the default value", func() {
				So(v, ShouldEqual, 42)
			})
		})

		Convey("When putting a value into the pool", func() {
			p.Put(99)

			Convey("And getting again", func() {
				v := p.Get()

				Convey("Then it should return the same value", func() {
					So(v, ShouldEqual, 99)
				})
			})
		})
	})

	Convey("Given a pool of strings", t, func() {
		p := New(func() string { return "hello" })

		Convey("When getting a value", func() {
			v := p.Get()

			Convey("Then it should return the default string", func() {
				So(v, ShouldEqual, "hello")
			})
		})
	})

	Convey("Given a pool of *strings", t, func() {
		p := New(func() *string { return new(string) })

		Convey("When getting a value", func() {
			v := p.Get()

			Convey("Then it should return the default string", func() {
				So(*v, ShouldEqual, "")
			})
		})

		Convey("When putting a value into the pool", func() {
			v := p.Get()
			p.Put(v)

			Convey("And getting again", func() {
				v := p.Get()

				Convey("Then it should return the default string", func() {
					So(v, ShouldNotBeNil)
					So(*v, ShouldEqual, "")
				})
			})
		})
	})
	Convey("Given a pool of ints with concurrent access", t, func() {
		p := New(func() int { return 0 })
		var wg sync.WaitGroup
		const goroutines = 100
		results := make(chan int, goroutines)

		Convey("When multiple goroutines put and get values", func() {
			for i := 0; i < goroutines; i++ {
				wg.Go(func() {
					p.Put(i)
					results <- p.Get()
				})
			}
			wg.Wait()
			close(results)

			Convey("Then all goroutines should complete without panic", func() {
				count := 0
				for v := range results {
					So(v, ShouldBeBetweenOrEqual, 0, goroutines-1) // либо из пула, либо новый
					count++
				}
				So(count, ShouldEqual, goroutines)
			})
		})
	})
}
