package wp

type worker[T any] struct {
	ch chan T
}
