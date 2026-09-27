package types

type messageError struct {
	message string
	cause   error
}

func (e messageError) Error() string {
	return e.message
}

func (e messageError) Unwrap() error {
	return e.cause
}

// WithMessage returns a user-facing error while preserving the underlying cause.
func WithMessage(message string, cause error) error {
	return messageError{message: message, cause: cause}
}
