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

type MultiReturns struct {
	Val interface{}
	Err error
}

func isSlice(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice
}

func TestTake(t *testing.T) {
	t.Run("Should works as expected", func(t *testing.T) {
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

		equal := reflect.DeepEqual([]int{2, 1, 3}, rs)
		if !equal {
			t.Errorf("Execution order is wrong, gets: %+v\n", rs)
		}
	})

	t.Run("Should limit results", func(t *testing.T) {
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

		equal := reflect.DeepEqual([]int{2, 1}, rs)
		if !equal {
			t.Errorf("Execution order is wrong, gets: %+v\n", rs)
		}
	})

	t.Run("Should cancel tasks when root context canceled",
		func(t *testing.T) {
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
		},
	)

	// FIXME:
	t.Run("Sub tasks without delay", func(t *testing.T) {
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

		// TODO: incomplete test
	})

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

	equal := reflect.DeepEqual([]int{1, 2, 3}, rs)
	if !equal {
		t.Errorf("Execution order is wrong, gets: %+v\n", rs)
	}
}

func TestAllWhenOutOfOrder(t *testing.T) {
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
				time.Sleep(300 * time.Millisecond)
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				<-time.After(100 * time.Millisecond)
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

	equal := reflect.DeepEqual([]int{1, 2, 3}, rs)
	if !equal {
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

	equal := reflect.DeepEqual([]int{1, 3}, rs)
	if !equal {
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

func TestRetry(t *testing.T) {
	t.Run("Should returns results on success", func(t *testing.T) {
		t.Parallel()

		rootCtx := context.Background()
		ch := make(chan MultiReturns)
		go func() {
			r, err := Retry(
				rootCtx,
				3,
				func(ctx context.Context) (interface{}, error) {
					time.Sleep(200 * time.Millisecond)
					return 1, nil
				},
			)

			ch <- MultiReturns{r, err}
			close(ch)
		}()

		r := <-ch
		if r.Err != nil {
			t.Errorf("Gets an error: %v\n", r.Err)
		}

		if r.Val != 1 {
			t.Errorf("Should gets `1`, gets: %+v instead\n", r.Val)
		}
	})

	t.Run("Should returns results after failing several times", func(t *testing.T) {
		t.Parallel()

		var times int32
		rootCtx := context.Background()
		ch := make(chan MultiReturns)
		go func() {
			r, err := Retry(
				rootCtx,
				3,
				func(ctx context.Context) (interface{}, error) {
					atomic.AddInt32(&times, 1)
					if atomic.LoadInt32(&times) >= 2 {
						return 1, nil
					}
					return nil, fmt.Errorf("err")
				},
			)

			ch <- MultiReturns{r, err}
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
	})

	t.Run("Should keep retrying until the limit is reached", func(t *testing.T) {
		t.Parallel()

		var times int32
		rootCtx := context.Background()
		ch := make(chan MultiReturns)
		go func() {

			r, err := Retry(
				rootCtx,
				3,
				func(ctx context.Context) (interface{}, error) {
					atomic.AddInt32(&times, 1)
					return nil, fmt.Errorf("err")
				},
			)

			ch <- MultiReturns{r, err}
			close(ch)
		}()

		r := <-ch
		if r.Err == nil {
			t.Errorf("Should gets an error")
		}

		if times != 3 {
			t.Errorf("Should tried 3 times, instead of: %+v\n", times)
		}
	})
}
