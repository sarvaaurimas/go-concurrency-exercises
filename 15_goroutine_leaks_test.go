package concurrency

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestLeakyFunction(t *testing.T) {
	LeakyFunction()
}

func TestFetchFirst(t *testing.T) {
	FetchFirst([]func() string{
		func() string { return "hello" },
		func() string { return "hello" },
		func() string { return "hello" },
	})
}
