# Deprecation notice

**As of 2022 I am not maintaining this tool** because there are better options
for examining Go benchmark output. In particular, [benchstat] is really all you
need.

Benchmark results are normally used by comparing before and after results which
is exactly what benchstat does best. But even if you are looking at a single
benchmark result (or, ideally, the result of several consecutive runs),
benchstat can format the output in a reasonable way that's nice to read.

Here's an example which compares the raw output of `go test -bench` to
`benchstat` for a single benchmark result (with five iterations):

```
$ go test -bench Sum64String -count 5 -benchtime 500ms | tee bench.txt
goos: linux
goarch: amd64
pkg: github.com/cespare/xxhash/v2
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkSum64String/4B-12      154079364                3.884 ns/op    1029.97 MB/s
BenchmarkSum64String/4B-12      154141522                3.856 ns/op    1037.46 MB/s
BenchmarkSum64String/4B-12      148497795                3.897 ns/op    1026.56 MB/s
BenchmarkSum64String/4B-12      154711791                3.866 ns/op    1034.71 MB/s
BenchmarkSum64String/4B-12      154970107                3.880 ns/op    1030.82 MB/s
BenchmarkSum64String/16B-12             100000000                5.112 ns/op    3129.95 MB/s
BenchmarkSum64String/16B-12             100000000                5.116 ns/op    3127.24 MB/s
BenchmarkSum64String/16B-12             100000000                5.141 ns/op    3112.24 MB/s
BenchmarkSum64String/16B-12             100000000                5.505 ns/op    2906.60 MB/s
BenchmarkSum64String/16B-12             100000000                5.188 ns/op    3084.00 MB/s
BenchmarkSum64String/100B-12            37342638                13.87 ns/op     7212.16 MB/s
BenchmarkSum64String/100B-12            43371970                13.87 ns/op     7208.80 MB/s
BenchmarkSum64String/100B-12            43732201                13.80 ns/op     7248.44 MB/s
BenchmarkSum64String/100B-12            42654915                13.96 ns/op     7163.70 MB/s
BenchmarkSum64String/100B-12            42779256                13.93 ns/op     7178.98 MB/s
BenchmarkSum64String/4KB-12              2333538               257.0 ns/op      15565.49 MB/s
BenchmarkSum64String/4KB-12              2461215               243.0 ns/op      16462.78 MB/s
BenchmarkSum64String/4KB-12              2495049               245.0 ns/op      16329.05 MB/s
BenchmarkSum64String/4KB-12              2440935               241.5 ns/op      16561.39 MB/s
BenchmarkSum64String/4KB-12              2482737               243.2 ns/op      16447.73 MB/s
BenchmarkSum64String/10MB-12                 848            698600 ns/op        14314.35 MB/s
BenchmarkSum64String/10MB-12                 843            715944 ns/op        13967.58 MB/s
BenchmarkSum64String/10MB-12                 853            697761 ns/op        14331.56 MB/s
BenchmarkSum64String/10MB-12                 838            700098 ns/op        14283.71 MB/s
BenchmarkSum64String/10MB-12                 846            701703 ns/op        14251.05 MB/s
PASS
ok      github.com/cespare/xxhash/v2    18.909s
$ benchstat bench.txt
name                 time/op
Sum64String/4B-12      3.88ns ± 1%
Sum64String/16B-12     5.21ns ± 6%
Sum64String/100B-12    13.9ns ± 1%
Sum64String/4KB-12      246ns ± 4%
Sum64String/10MB-12     703µs ± 2%

name                 speed
Sum64String/4B-12    1.03GB/s ± 1%
Sum64String/16B-12   3.07GB/s ± 5%
Sum64String/100B-12  7.20GB/s ± 1%
Sum64String/4KB-12   16.3GB/s ± 4%
Sum64String/10MB-12  14.2GB/s ± 2%
```

[benchstat]: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat

---

# Prettybench

A tool for transforming `go test`'s benchmark output a bit to make it nicer for humans.

## Problem

Go benchmarks are great, particularly when used in concert with benchcmp. But
the output can be a bit hard to read:

![before](/screenshots/before.png)

## Solution

    $ go install github.com/cespare/prettybench@latest
    $ go test -bench . | prettybench

![after](/screenshots/after.png)

* Column headers
* Columns are aligned
* Time output is adjusted to convenient units

## Notes

* Right now the units for the time are chosen based on the smallest value in the
  column.
* Prettybench has to buffer all the rows of output before it can print them (for
  column formatting), so you won't see intermediate progress. If you want to see
  that too, you could tee your output so that you see the unmodified version as
  well. If you do this, you'll want to use the prettybench's `-no-passthrough`
  flag so it doesn't print all the other lines (because then they'd be printed
  twice):

        $ go test -bench . | tee >(prettybench -no-passthrough)

## To Do (maybe)

* Handle benchcmp output
* Change the units for non-time columns as well (these are generally OK though).
