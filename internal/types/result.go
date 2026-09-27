package types

// Result represents either a value or an error.
type Result[T any] struct {
	value T
	err   error
}

// Success creates a successful result.
func Success[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Failure creates a failed result. err must not be nil.
func Failure[T any](err error) Result[T] {
	if err == nil {
		panic("types.Failure called with nil error")
	}

	return Result[T]{err: err}
}

// IsSuccess reports whether the result contains a value.
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// Value returns the result value. It panics if the result is a failure.
func (r Result[T]) Value() T {
	if r.err != nil {
		panic("Value called on failed result")
	}

	return r.value
}

// Error returns the result error, or nil for a successful result.
func (r Result[T]) Error() error {
	return r.err
}
