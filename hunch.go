// Package hunch provides functions like: `All`, `First`, `Retry`, `Waterfall` etc., that makes asynchronous flow control more intuitive.
package hunch

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Executable represents a singular logic block.
// It can be used with several functions.
type Executable func(context.Context) (interface{}, error)

// ExecutableInSequence represents one of a sequence of logic blocks.
type ExecutableInSequence func(context.Context, interface{}) (interface{}, error)

// IndexedValue stores the output of Executables,
// along with the index of the source Executable for ordering.
type IndexedValue struct {
	Index int
	Value interface{}
}

// IndexedExecutableOutput stores both output and error values from a Excetable.
type IndexedExecutableOutput struct {
	Value IndexedValue
	Err   error
}

func pluckVals(iVals []IndexedValue) []interface{} {
	vals := []interface{}{}
	for _, val := range iVals {
		vals = append(vals, val.Value)
	}

	return vals
}

func sortIdxVals(iVals []IndexedValue) []IndexedValue {
	sorted := make([]IndexedValue, len(iVals))
	copy(sorted, iVals)
	sort.SliceStable(
		sorted,
		func(i, j int) bool {
			return sorted[i].Index < sorted[j].Index
		},
	)

	return sorted
}

// Take returns the first `num` values outputted by the Executables.
func Take(parentCtx context.Context, num int, execs ...Executable) ([]interface{}, error) {
	execCount := len(execs)

	if num > execCount {
		num = execCount
	}

	// Create a new sub-context for possible cancelation.
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	output := make(chan IndexedExecutableOutput, 1)
	go runExecs(ctx, output, execs)

	fail := make(chan error, 1)
	success := make(chan []IndexedValue, 1)
	go takeUntilEnough(fail, success, min(len(execs), num), output)

	select {

	case <-parentCtx.Done():
		// Stub comment to fix a test coverage bug.
		return nil, parentCtx.Err()

	case err := <-fail:
		cancel()
		if parentCtxErr := parentCtx.Err(); parentCtxErr != nil {
			return nil, parentCtxErr
		}
		return nil, err

	case uVals := <-success:
		cancel()
		return pluckVals(uVals), nil
	}
}

func runExecs(ctx context.Context, output chan<- IndexedExecutableOutput, execs []Executable) {
	var wg sync.WaitGroup
	for i, exec := range execs {
		wg.Add(1)

		go func(i int, exec Executable) {
			val, err := exec(ctx)
			if err != nil {
				output <- IndexedExecutableOutput{
					IndexedValue{i, nil},
					err,
				}
				wg.Done()
				return
			}

			output <- IndexedExecutableOutput{
				IndexedValue{i, val},
				nil,
			}
			wg.Done()
		}(i, exec)
	}

	wg.Wait()
	close(output)
}

func takeUntilEnough(fail chan error, success chan []IndexedValue, num int, output chan IndexedExecutableOutput) {
	uVals := make([]IndexedValue, num)

	enough := false
	outputCount := 0
	for r := range output {
		if enough {
			continue
		}

		if r.Err != nil {
			enough = true
			fail <- r.Err
			continue
		}

		uVals[outputCount] = r.Value
		outputCount++

		if outputCount == num {
			enough = true
			success <- uVals
			continue
		}
	}
}

// All returns all the outputs from all Executables, order guaranteed.
func All(parentCtx context.Context, execs ...Executable) ([]interface{}, error) {
	// Create a new sub-context for possible cancelation.
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	output := make(chan IndexedExecutableOutput, 1)
	go runExecs(ctx, output, execs)

	fail := make(chan error, 1)
	success := make(chan []IndexedValue, 1)
	go takeUntilEnough(fail, success, len(execs), output)

	select {

	case <-parentCtx.Done():
		// Stub comment to fix a test coverage bug.
		return nil, parentCtx.Err()

	case err := <-fail:
		cancel()
		if parentCtxErr := parentCtx.Err(); parentCtxErr != nil {
			return nil, parentCtxErr
		}
		return nil, err

	case uVals := <-success:
		cancel()
		return pluckVals(sortIdxVals(uVals)), nil
	}
}

/*
Last returns the last `num` values outputted by the Executables.
*/
func Last(parentCtx context.Context, num int, execs ...Executable) ([]interface{}, error) {
	execCount := len(execs)
	if num > execCount {
		num = execCount
	}
	start := execCount - num

	vals, err := Take(parentCtx, execCount, execs...)
	if err != nil {
		return nil, err
	}

	return vals[start:], err
}

// MaxRetriesExceededError stores how many times did an Execution run before exceeding the limit.
// The retries field holds the value.
type MaxRetriesExceededError struct {
	retries int
}

func (err MaxRetriesExceededError) Error() string {
	var word string
	switch err.retries {
	case 0:
		word = "infinity"
	case 1:
		word = "1 time"
	default:
		word = fmt.Sprintf("%v times", err.retries)
	}

	return fmt.Sprintf("Max retries exceeded (%v).\n", word)
}

// Retry attempts to get a value from an Executable instead of an Error.
// It will keeps re-running the Executable when failed no more than `retries` times.
// Also, when the parent Context canceled, it returns the `Err()` of it immediately.
func Retry(parentCtx context.Context, retries int, fn Executable) (interface{}, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	c := 0
	fail := make(chan error, 1)
	success := make(chan interface{}, 1)

	for {
		go func() {
			val, err := fn(ctx)
			if err != nil {
				fail <- err
				return
			}
			success <- val
		}()

		select {
		//
		case <-parentCtx.Done():
			// Stub comment to fix a test coverage bug.
			return nil, parentCtx.Err()

		case <-fail:
			if parentCtxErr := parentCtx.Err(); parentCtxErr != nil {
				return nil, parentCtxErr
			}

			c++
			if retries == 0 || c < retries {
				continue
			}
			return nil, MaxRetriesExceededError{c}

		case val := <-success:
			return val, nil
		}
	}
}

// Waterfall runs `ExecutableInSequence`s one by one,
// passing previous result to next Executable as input.
// When an error occurred, it stop the process then returns the error.
// When the parent Context canceled, it returns the `Err()` of it immediately.
func Waterfall(parentCtx context.Context, execs ...ExecutableInSequence) (interface{}, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var lastVal interface{}
	execCount := len(execs)
	i := 0
	fail := make(chan error, 1)
	success := make(chan interface{}, 1)

	for {
		go func() {
			val, err := execs[i](ctx, lastVal)
			if err != nil {
				fail <- err
				return
			}
			success <- val
		}()

		select {

		case <-parentCtx.Done():
			// Stub comment to fix a test coverage bug.
			return nil, parentCtx.Err()

		case err := <-fail:
			if parentCtxErr := parentCtx.Err(); parentCtxErr != nil {
				return nil, parentCtxErr
			}

			return nil, err

		case val := <-success:
			lastVal = val
			i++
			if i == execCount {
				return val, nil
			}

			continue
		}
	}
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
