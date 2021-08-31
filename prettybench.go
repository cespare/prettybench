package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	bench "golang.org/x/tools/benchmark/parse"
)

// ----------------------------------------------------------------------------
//  Main Function
// ----------------------------------------------------------------------------

func main() {
	flag.Parse()

	currentBenchmark := &BenchOutputGroup{}
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		text := scanner.Text()
		line, err := ParseLine(text)

		switch err {
		case errNotBenchLine:
			if okLineMatcher.MatchString(text) {
				fmt.Print(currentBenchmark)
				currentBenchmark = &BenchOutputGroup{}
			}

			if !*noPassthrough {
				fmt.Println(text)
			}
		case nil:
			currentBenchmark.AddLine(line)
		default:
			fmt.Fprintln(os.Stderr, "prettybench unrecognized line:")
			fmt.Println(text)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ----------------------------------------------------------------------------
//  Variables
// ----------------------------------------------------------------------------

var (
	noPassthrough    = flag.Bool("no-passthrough", false, "Don't print non-benchmark lines")
	benchLineMatcher = regexp.MustCompile(`^Benchmark.*\t.*\d+`)
	okLineMatcher    = regexp.MustCompile(`^ok\s`)
	errNotBenchLine  = errors.New("not a bench line")
)

// ----------------------------------------------------------------------------
//  Types
// ----------------------------------------------------------------------------

type BenchOutputGroup struct {
	Lines []*bench.Benchmark
	// Columns which are in use
	Measured int
}

type Table struct {
	MaxLengths []int
	Cells      [][]string
}

// ----------------------------------------------------------------------------
//  Methods of BenchOutputGroup
// ----------------------------------------------------------------------------

// AddLine appends line to Lines field.
func (g *BenchOutputGroup) AddLine(line *bench.Benchmark) {
	g.Lines = append(g.Lines, line)
	g.Measured |= line.Measured
}

// String is a stringer of BenchOutputGroup type.
func (g *BenchOutputGroup) String() string {
	if len(g.Lines) == 0 {
		return ""
	}

	columnNames := []string{"benchmark", "iter", "time/iter"}

	if (g.Measured & bench.MBPerS) > 0 {
		columnNames = append(columnNames, "throughput")
	}

	if (g.Measured & bench.AllocedBytesPerOp) > 0 {
		columnNames = append(columnNames, "bytes alloc")
	}

	if (g.Measured & bench.AllocsPerOp) > 0 {
		columnNames = append(columnNames, "allocs")
	}

	table := tabulate(g, columnNames)

	table.MaxLengths = findMaxLengths(columnNames, table.Cells)

	return formatTableCells(table.Cells, table.MaxLengths)
}

// TimeFormatFunc uniforms the time unit to ns/μs/ms/s.
func (g *BenchOutputGroup) TimeFormatFunc() func(float64) string {
	// Find the smallest time
	smallest := g.Lines[0].NsPerOp
	for _, line := range g.Lines[1:] {
		if line.NsPerOp < smallest {
			smallest = line.NsPerOp
		}
	}

	switch {
	case smallest < float64(10000*time.Nanosecond):
		return func(ns float64) string {
			return fmt.Sprintf("%.2f ns/op", ns)
		}
	case smallest < float64(time.Millisecond):
		return func(ns float64) string {
			return fmt.Sprintf("%.2f μs/op", ns/1000)
		}
	case smallest < float64(10*time.Second):
		return func(ns float64) string {
			return fmt.Sprintf("%.2f ms/op", (ns / 1e6))
		}
	default:
		return func(ns float64) string {
			return fmt.Sprintf("%.2f s/op", ns/1e9)
		}
	}
}

// ----------------------------------------------------------------------------
//  Functions
// ----------------------------------------------------------------------------

func tabulate(g *BenchOutputGroup, columnNames []string) *Table {
	table := &Table{Cells: [][]string{columnNames}}
	underlines := make([]string, len(columnNames))

	for _, name := range columnNames {
		underlines = append(underlines, strings.Repeat("-", len(name)))
	}

	table.Cells = append(table.Cells, underlines)
	timeFormatFunc := g.TimeFormatFunc()

	for _, line := range g.Lines {
		row := []string{line.Name, FormatIterations(line.N), timeFormatFunc(line.NsPerOp)}
		if (g.Measured & bench.MBPerS) > 0 {
			row = append(row, FormatMegaBytesPerSecond(line))
		}

		if (g.Measured & bench.AllocedBytesPerOp) > 0 {
			row = append(row, FormatBytesAllocPerOp(line))
		}

		if (g.Measured & bench.AllocsPerOp) > 0 {
			row = append(row, FormatAllocsPerOp(line))
		}

		table.Cells = append(table.Cells, row)
	}

	return table
}

func findMaxLengths(colNames []string, tableCells [][]string) (tableMaxLengths []int) {
	for i := range colNames {
		maxLength := 0
		for _, row := range tableCells {
			if len(row[i]) > maxLength {
				maxLength = len(row[i])
			}
		}

		tableMaxLengths = append(tableMaxLengths, maxLength)
	}

	return tableMaxLengths
}

func formatTableCells(tableCells [][]string, tableMaxLengths []int) string {
	var buf bytes.Buffer

	for _, row := range tableCells {
		for i, cell := range row {
			format := getFormat(i, len(row))
			fmt.Fprintf(&buf, fmt.Sprintf(format, tableMaxLengths[i]), cell)
		}

		fmt.Fprint(&buf, "\n")
	}

	return buf.String()
}

func getFormat(rowNum int, rowLen int) (format string) {
	switch rowNum {
	case 0:
		format = "%%-%ds   "
	case rowLen - 1:
		format = "%%%ds"
	default:
		format = "%%%ds   "
	}

	return format
}

func FormatAllocsPerOp(l *bench.Benchmark) string {
	if (l.Measured & bench.AllocsPerOp) == 0 {
		return ""
	}

	return fmt.Sprintf("%d allocs/op", l.AllocsPerOp)
}

func FormatBytesAllocPerOp(l *bench.Benchmark) string {
	if (l.Measured & bench.AllocedBytesPerOp) == 0 {
		return ""
	}

	return fmt.Sprintf("%d B/op", l.AllocedBytesPerOp)
}

func FormatIterations(iter int) string {
	return strconv.FormatInt(int64(iter), 10)
}

func FormatMegaBytesPerSecond(l *bench.Benchmark) string {
	if (l.Measured & bench.MBPerS) == 0 {
		return ""
	}

	return fmt.Sprintf("%.2f MB/s", l.MBPerS)
}

func ParseLine(line string) (*bench.Benchmark, error) {
	if !benchLineMatcher.MatchString(line) {
		return nil, errNotBenchLine
	}

	fields := strings.Split(line, "\t")

	if len(fields) < 3 {
		return nil, errNotBenchLine
	}

	return bench.ParseLine(line)
}
