package wp

type stack[T any] []T

func NewStack[T any](size int) stack[T] {
	return stack[T](make([]T, 0, size))
}

func (s *stack[T]) Push(v T) {
	*s = append(*s, v)
}

func (s *stack[T]) Pop() T {
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func (s *stack[T]) Size() int {
	return len(*s)
}
