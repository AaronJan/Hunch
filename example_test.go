package hunch_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronjan/hunch"
)

func ExampleAll() {
	ctx := context.Background()
	r, err := hunch.All(
		ctx,
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(300 * time.Millisecond)
			return 1, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(200 * time.Millisecond)
			return 2, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return 3, nil
		},
	)

	fmt.Println(r, err)
	// Output:
	// [1 2 3] <nil>
}

func ExampleTake() {
	ctx := context.Background()
	r, err := hunch.Take(
		ctx,
		// Only need the first 2 values.
		2,
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(300 * time.Millisecond)
			return 1, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(200 * time.Millisecond)
			return 2, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return 3, nil
		},
	)

	fmt.Println(r, err)
	// Output:
	// [3 2] <nil>
}

func ExampleLast() {
	ctx := context.Background()
	r, err := hunch.Last(
		ctx,
		// Only need the last 2 values.
		2,
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(300 * time.Millisecond)
			return 1, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(200 * time.Millisecond)
			return 2, nil
		},
		func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return 3, nil
		},
	)

	fmt.Println(r, err)
	// Output:
	// [2 1] <nil>
}

func ExampleWaterfall() {
	ctx := context.Background()
	r, err := hunch.Waterfall(
		ctx,
		func(ctx context.Context, n interface{}) (interface{}, error) {
			return 1, nil
		},
		func(ctx context.Context, n interface{}) (interface{}, error) {
			return n.(int) + 1, nil
		},
		func(ctx context.Context, n interface{}) (interface{}, error) {
			return n.(int) + 1, nil
		},
	)

	fmt.Println(r, err)
	// Output:
	// 3 <nil>
}

func ExampleRetry() {
	count := 0
	getStuffFromAPI := func() (int, error) {
		if count == 5 {
			return 1, nil
		}
		count++

		return 0, fmt.Errorf("timeout")
	}

	ctx := context.Background()
	r, err := hunch.Retry(
		ctx,
		10,
		func(ctx context.Context) (interface{}, error) {
			rs, err := getStuffFromAPI()

			return rs, err
		},
	)

	fmt.Println(r, err, count)
	// Output:
	// 1 <nil> 5
}
