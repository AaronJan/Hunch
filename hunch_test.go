package hunch

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

type AppError struct {
}

func (err AppError) Error() string {
	return "app error!"
}

type MultiReturns[T any] struct {
	Val []T
	Err error
}

type SingleReturn[T any] struct {
	Val T
	Err error
}

func isSlice[T any](v T) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice
}

func TestTake_ShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	equal := reflect.DeepEqual([]int{2, 1, 3}, r.Val)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", r.Val)
	}
}

func TestTake_ShouldJustTakeAll(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Take(
			rootCtx,
			100,
			func(ctx context.Context) (int, error) {
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	if len(r.Val) != 3 {
		t.Errorf("Should return 3 results, but gets: %+v\n", r.Val)
	}
}

func TestTake_ShouldLimitResults(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Take(
			rootCtx,
			2,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch

	equal := reflect.DeepEqual([]int{2, 1}, r.Val)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", r.Val)
	}
}

func TestTake_ShouldCancelWhenOneExecutableReturnedError(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 0, AppError{}
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
}

func TestTake_ShouldCancelWhenRootCanceled(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				select {
				case <-ctx.Done():
					return 0, AppError{}
				case <-time.After(300 * time.Millisecond):
					return 3, nil
				}
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
}

func TestAll_ShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	equal := reflect.DeepEqual([]int{1, 2, 3}, r.Val)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", r.Val)
	}
}

func TestAll_WhenOutOfOrder(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(300 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				<-time.After(100 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	equal := reflect.DeepEqual([]int{1, 2, 3}, r.Val)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", r.Val)
	}
}

func TestAll_WhenRootCtxCanceled(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				select {
				case <-ctx.Done():
					return 0, AppError{}
				case <-time.After(300 * time.Millisecond):
					return 3, nil
				}
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
}

func TestAll_WhenAnySubFunctionFailed(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan MultiReturns[int])
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(300 * time.Millisecond)
				return 0, AppError{}
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
	if r.Err.Error() != "app error!" {
		t.Errorf("Should returns an AppError, gets \"%v\"\n", r.Err.Error())
	}
}

func TestRetry_WithNoFailure(t *testing.T) {
	t.Parallel()

	times := 0
	expect := 1

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan SingleReturn[int])
	go func() {
		r, err := Retry(
			rootCtx,
			10,
			func(ctx context.Context) (int, error) {
				times++

				time.Sleep(100 * time.Millisecond)
				return expect, nil
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	val := r.Val
	if val != expect {
		t.Errorf("Unmatched value: %v\n", r.Val)
	}
	if times != 1 {
		t.Errorf("Should execute only once, gets: %v\n", times)
	}
}

func TestLast_ShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns[int])
	go func() {
		r, err := Last(
			rootCtx,
			2,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (int, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	equal := reflect.DeepEqual([]int{1, 3}, r.Val)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", r.Val)
	}
}

func TestWaterfall_ShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan SingleReturn[int])
	go func() {
		r, err := Waterfall(
			rootCtx,
			func(ctx context.Context, n int) (int, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context, n int) (int, error) {
				time.Sleep(100 * time.Millisecond)
				n = n + 1
				return n, nil
			},
			func(ctx context.Context, n int) (int, error) {
				time.Sleep(100 * time.Millisecond)
				n = n + 1
				return n, nil
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	if r.Val != 3 {
		t.Errorf("Expect \"3\", but gets: \"%v\"\n", r.Val)
	}
}

func TestRetry_ShouldReturnsOnSuccess(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan SingleReturn[int])
	go func() {
		r, err := Retry(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	if r.Val != 1 {
		t.Errorf("Should gets `1`, gets: %+v instead\n", r.Val)
	}
}

func TestRetry_ShouldReturnsAfterFailingSeveralTimes(t *testing.T) {
	t.Parallel()

	var times int32
	rootCtx := context.Background()
	ch := make(chan SingleReturn[int])
	go func() {
		r, err := Retry(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				atomic.AddInt32(&times, 1)
				if atomic.LoadInt32(&times) >= 2 {
					return 1, nil
				}
				return 0, fmt.Errorf("err")
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	if times != 2 {
		t.Errorf("Should tried 2 times, instead of %v\n", times)
	}

	if r.Val != 1 {
		t.Errorf("Should gets `1`, gets: %+v instead\n", r.Val)
	}
}

func TestRetry_ShouldKeepRetrying(t *testing.T) {
	t.Parallel()

	var times int32
	rootCtx := context.Background()
	ch := make(chan SingleReturn[int])
	go func() {

		r, err := Retry(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				atomic.AddInt32(&times, 1)
				return 0, fmt.Errorf("err")
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err == nil {
		t.Errorf("Should gets an error")
	}

	if times != 3 {
		t.Errorf("Should tried 3 times, instead of: %+v\n", times)
	}
}

func TestRetry_WhenRootCtxCanceled(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan SingleReturn[int])
	go func() {
		r, err := Retry(
			rootCtx,
			3,
			func(ctx context.Context) (int, error) {
				time.Sleep(50 * time.Millisecond)
				return 0, fmt.Errorf("err")
			},
		)

		ch <- SingleReturn[int]{r, err}
		close(ch)
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	r := <-ch
	if r.Err == nil {
		t.Errorf("Should gets an error")
	}
}

func TestMin(t *testing.T) {
	t.Parallel()

	if min(1, 2) != 1 {
		t.Errorf("Should returns 1")
	}
	if min(-1, 2) != -1 {
		t.Errorf("Should returns -1")
	}
	if min(54321, 12345) != 12345 {
		t.Errorf("Should returns 12345")
	}
}
