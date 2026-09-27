package types

import (
	"errors"
	"testing"
)

func TestSuccess(t *testing.T) {
	result := Success("value")

	if !result.IsSuccess() {
		t.Fatal("expected success")
	}
	if result.Value() != "value" {
		t.Fatalf("Value() = %q, want %q", result.Value(), "value")
	}
	if result.Error() != nil {
		t.Fatalf("Error() = %v, want nil", result.Error())
	}
}

func TestFailure(t *testing.T) {
	want := errors.New("failed")
	result := Failure[string](want)

	if result.IsSuccess() {
		t.Fatal("expected failure")
	}
	if !errors.Is(result.Error(), want) {
		t.Fatalf("Error() = %v, want %v", result.Error(), want)
	}
}

func TestFailurePanicsForNilError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Failure did not panic")
		}
	}()

	Failure[string](nil)
}

func TestValuePanicsForFailure(t *testing.T) {
	result := Failure[string](errors.New("failed"))

	defer func() {
		if recover() == nil {
			t.Fatal("Value did not panic")
		}
	}()

	result.Value()
}

func TestWithMessagePreservesCause(t *testing.T) {
	cause := errors.New("technical failure")
	err := WithMessage("Something went wrong.\n\nTry again.", cause)

	if err.Error() != "Something went wrong.\n\nTry again." {
		t.Fatalf("Error() = %q", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("error does not wrap %v", cause)
	}
}
