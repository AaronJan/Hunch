<a href="https://github.com/AaronJan/Hunch">
	<img width="160" src="https://user-images.githubusercontent.com/4630940/59684078-ea9d8c80-920b-11e9-9c99-85051dcb8a04.jpg" alt="Housekeeper" title="Hunch" align="right"/>
</a>

![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/aaronjan/hunch.svg)
[![Build status](https://ci.appveyor.com/api/projects/status/co97gxwpcewrebip/branch/master?svg=true)](https://ci.appveyor.com/project/AaronJan/hunch/branch/master)
[![codecov](https://codecov.io/gh/AaronJan/Hunch/branch/master/graph/badge.svg)](https://codecov.io/gh/AaronJan/Hunch)
[![Go Report Card](https://goreportcard.com/badge/github.com/aaronjan/hunch)](https://goreportcard.com/report/github.com/aaronjan/hunch)
![GitHub](https://img.shields.io/github/license/aaronjan/hunch.svg)
[![GoDoc](https://godoc.org/github.com/aaronjan/hunch?status.svg)](https://godoc.org/github.com/aaronjan/hunch)

# Hunch

Hunch provides functions like: `All`, `First`, `Retry`, `Waterfall` etc., that makes asynchronous flow control more intuitive.

## About Hunch

Go have several concurrency patterns, here're some articles:

* https://blog.golang.org/pipelines
* https://blog.golang.org/context
* https://blog.golang.org/go-concurrency-patterns-timing-out-and
* https://blog.golang.org/advanced-go-concurrency-patterns

But nowadays, using the `context` package is the most powerful pattern.

So base on `context`, Hunch provides functions that can help you deal with complex asynchronous logics with ease.

## Usage

### Installation

#### `go get`

```shell
$ go get -u -v github.com/aaronjan/hunch
```

#### `go mod` (Recommended)

```go
import "github.com/aaronjan/hunch"
```

```shell
$ go mod tidy
```

### Types

```go
type Executable func(context.Context) (interface{}, error)

type ExecutableInSequence func(context.Context, interface{}) (interface{}, error)
```

### API

#### All

```go
func All(parentCtx context.Context, execs ...Executable) ([]interface{}, error) 
```

All returns all the outputs from all Executables, order guaranteed.

##### Examples

```go
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
// output:
// [1 2 3] <nil>
```

#### Take

```go
func Take(parentCtx context.Context, num int, execs ...Executable) ([]interface{}, error)
```

Take returns the first `num` values outputted by the Executables.

##### Examples

```go
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
// output:
// [3 2] <nil>
```

#### Last

```go
func Last(parentCtx context.Context, num int, execs ...Executable) ([]interface{}, error)
```

Last returns the last `num` values outputted by the Executables.

##### Examples

```go
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
// output:
// [2 1] <nil>
```

#### Waterfall

```go
func Waterfall(parentCtx context.Context, execs ...ExecutableInSequence) (interface{}, error)
```

Waterfall runs `ExecutableInSequence`s one by one, passing previous result to next Executable as input. When an error occurred, it stop the process then returns the error. When the parent Context canceled, it returns the `Err()` of it immediately.

##### Examples

```go
r, err := hunch.Waterfall(
    ctx,
    func(ctx context.Context, n interface{}) (interface{}, error) {
        return 1, nil
    },
    func(ctx context.Context, n interface{}) (interface{}, error) {
        return n + 1, nil
    },
    func(ctx context.Context, n interface{}) (interface{}, error) {
        return n + 1, nil
    },
)

fmt.Println(r, err)
// output:
// 3 <nil>
```

#### Retry

```go
func Retry(parentCtx context.Context, retries int, fn Executable) (interface{}, error)
```

Retry attempts to get a value from an Executable instead of an Error. It will keeps re-running the Executable when failed no more than `retries` times. Also, when the parent Context canceled, it returns the `Err()` of it immediately.

##### Examples

```go
r, err := hunch.Retry(
    ctx,
    10,
    func(ctx context.Context) (interface{}, error) {
        rs, err := api.GetResults()

        return rs, err
    },
)
```

## Credits

Heavily inspired by [Async](https://github.com/caolan/async/) and [ReactiveX](http://reactivex.io/).

## Licence

[Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0)
