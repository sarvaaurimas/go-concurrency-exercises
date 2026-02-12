package concurrency

import (
	"slices"
	"sync/atomic"
	"testing"
)

func TestRaceCounter(t *testing.T) {
	expected := 1000
	count := RaceCounter()
	if count != expected {
		t.Errorf("Got %d, expected %d", count, expected)
	}
}

func TestRaceCounterFix(t *testing.T) {
	var expected atomic.Int64
	expected.Store(1000)
	count := RaceCounterFix()
	if *count != expected {
		t.Errorf("Got %v, expected %v", *count, expected)
	}
}

func TestRaceSlice(t *testing.T) {
	expected := []int{}
	for i := range 100 {
		expected = append(expected, i)
	}
	result := RaceSlice()
	if len(expected) != len(result) {
		t.Errorf("Got %d len, expected %d", len(result), len(expected))
	}
	if !slices.Equal(expected, result) {
		t.Errorf("Got %v, expected %v", result, expected)
	}
}

func TestRaceSliceFix(t *testing.T) {
	expected := []int{}
	for i := range 100 {
		expected = append(expected, i)
	}
	result := RaceSliceFix()
	if len(expected) != len(result) {
		t.Errorf("Got %d len, expected %d", len(result), len(expected))
	}
}

func TestRaceMap(t *testing.T) {
	expected := []int{}
	for i := range 100 {
		expected = append(expected, i)
	}
	result := RaceSlice()
	if len(expected) != len(result) {
		t.Errorf("Got %d len, expected %d", len(result), len(expected))
	}
	if !slices.Equal(expected, result) {
		t.Errorf("Got %v, expected %v", result, expected)
	}
}
