// Package generics demonstrates generic type extraction.
package generics

// Pair holds two values of potentially different types.
type Pair[A, B any] struct {
	First  A
	Second B
}

// Swap returns a new Pair with the values swapped.
func (p Pair[A, B]) Swap() Pair[B, A] {
	return Pair[B, A]{First: p.Second, Second: p.First}
}

// Ordered is a constraint for types that support comparison.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// Min returns the smaller of two ordered values.
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Map applies a function to every element of a slice.
func Map[T, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

// Filter returns elements that satisfy the predicate.
func Filter[T any](items []T, pred func(T) bool) []T {
	var result []T
	for _, item := range items {
		if pred(item) {
			result = append(result, item)
		}
	}
	return result
}
