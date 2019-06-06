package hunch

import (
	"context"

	"reflect"
	"testing"
	"time"

	"github.com/mb0/diff"
)

type AppError struct {
}

func (err AppError) Error() string {
	return "app error!"
}

type MultiReturns struct {
	Val interface{}
	Err error
}

func isSlice(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice
}

func TestTakeShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns)
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	rs := []int{}
	for _, v := range r.Val.([]interface{}) {
		rs = append(rs, v.(int))
	}

	changes := diff.Ints(rs, []int{2, 1, 3})
	if len(changes) != 0 {
		t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	}
}

func TestTakeShouldLimitResults(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns)
	go func() {
		r, err := Take(
			rootCtx,
			2,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	rs := []int{}
	for _, v := range r.Val.([]interface{}) {
		rs = append(rs, v.(int))
	}

	changes := diff.Ints(rs, []int{2, 1})
	if len(changes) != 0 {
		t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	}
}

func TestTakeWhenRootCtxCanceled(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan MultiReturns)
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				select {
				case <-ctx.Done():
					return nil, AppError{}
				case <-time.After(300 * time.Millisecond):
					return 3, nil
				}
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val.([]interface{})) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
}

func TestTakeWithoutDelay(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan MultiReturns)
	go func() {
		r, err := Take(
			rootCtx,
			3,
			func(ctx context.Context) (interface{}, error) {
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 3, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	rs := []int{}
	for _, v := range r.Val.([]interface{}) {
		rs = append(rs, v.(int))
	}

	// FIXME:
	// changes := diff.Ints(rs, []int{2, 1, 3})
	// if len(changes) != 0 {
	// 	t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	// }
}

func TestAllShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns)
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	rs := []int{}
	for _, v := range r.Val.([]interface{}) {
		rs = append(rs, v.(int))
	}

	changes := diff.Ints(rs, []int{1, 2, 3})
	if len(changes) != 0 {
		t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	}
}

func TestAllWhenRootCtxCanceled(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan MultiReturns)
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				select {
				case <-ctx.Done():
					return nil, AppError{}
				case <-time.After(300 * time.Millisecond):
					return 3, nil
				}
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val.([]interface{})) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
}

func TestAllWhenAnySubFunctionFailed(t *testing.T) {
	t.Parallel()

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan MultiReturns)
	go func() {
		r, err := All(
			rootCtx,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(300 * time.Millisecond)
				return nil, AppError{}
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if !isSlice(r.Val) || len(r.Val.([]interface{})) != 0 {
		t.Errorf("Return Value should be default, gets: \"%v\"\n", r.Val)
	}
	if r.Err == nil {
		t.Errorf("Should returns an Error, gets `nil`\n")
	}
	if r.Err.Error() != "app error!" {
		t.Errorf("Should returns an AppError, gets \"%v\"\n", r.Err.Error())
	}
}

func TestRetryWithNoFailure(t *testing.T) {
	t.Parallel()

	times := 0
	expect := 1

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan MultiReturns)
	go func() {
		r, err := Retry(
			rootCtx,
			10,
			func(ctx context.Context) (interface{}, error) {
				times++

				time.Sleep(100 * time.Millisecond)
				return expect, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	val, ok := r.Val.(int)
	if ok == false || val != expect {
		t.Errorf("Unmatched value: %v\n", r.Val)
	}
	if times != 1 {
		t.Errorf("Should execute only once, gets: %v\n", times)
	}
}

func TestLastShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns)
	go func() {
		r, err := Last(
			rootCtx,
			2,
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				<-time.After(300 * time.Millisecond)
				return 3, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	rs := []int{}
	for _, v := range r.Val.([]interface{}) {
		rs = append(rs, v.(int))
	}

	changes := diff.Ints(rs, []int{1, 3})
	if len(changes) != 0 {
		t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	}
}

func TestWaterfallShouldWorksAsExpected(t *testing.T) {
	t.Parallel()

	rootCtx := context.Background()
	ch := make(chan MultiReturns)
	go func() {
		r, err := Waterfall(
			rootCtx,
			func(ctx context.Context, n interface{}) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return 1, nil
			},
			func(ctx context.Context, n interface{}) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				n = n.(int) + 1
				return n, nil
			},
			func(ctx context.Context, n interface{}) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				n = n.(int) + 1
				return n, nil
			},
		)

		ch <- MultiReturns{r, err}
		close(ch)
	}()

	r := <-ch
	if r.Err != nil {
		t.Errorf("Gets an error: %v\n", r.Err)
	}

	if r.Val.(int) != 3 {
		t.Errorf("Expect \"3\", but gets: \"%v\"\n", r.Val)
	}
}
