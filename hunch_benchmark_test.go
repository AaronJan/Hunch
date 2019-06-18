package hunch

import (
	"context"
	"testing"
)

func BenchmarkTakeWithFiveExecsThatNeedsOne(b *testing.B) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Take(
			rootCtx,
			1,
			func(ctx context.Context) (interface{}, error) {
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 3, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 4, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 5, nil
			},
		)
	}
}

func BenchmarkTakeWithFiveExecsThatNeedsFive(b *testing.B) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Take(
			rootCtx,
			5,
			func(ctx context.Context) (interface{}, error) {
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 3, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 4, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 5, nil
			},
		)
	}
}

func BenchmarkAllWithFiveExecs(b *testing.B) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		All(
			rootCtx,
			func(ctx context.Context) (interface{}, error) {
				return 1, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 2, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 3, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 4, nil
			},
			func(ctx context.Context) (interface{}, error) {
				return 5, nil
			},
		)
	}
}
