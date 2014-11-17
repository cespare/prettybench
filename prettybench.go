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

	benchcmp "golang.org/x/tools/cmd/benchcmp"
)

var noPassthrough = flag.Bool("no-passthrough", false, "Don't print non-benchmark lines")

type BenchOutputGroup struct {
	Lines []*benchcmp.Bench
	// Columns which are in use
	MegaBytesPresent  bool
	BytesAllocPresent bool
	AllocsPresent     bool
}

type Table struct {
	MaxLengths []int
	Cells      [][]string
}

func (g *BenchOutputGroup) String() string {
	if len(g.Lines) == 0 {
		return ""
	}
	columnNames := []string{"benchmark", "iter", "time/iter"}
	if g.MegaBytesPresent {
		columnNames = append(columnNames, "throughput")
	}
	if g.BytesAllocPresent {
		columnNames = append(columnNames, "bytes alloc")
	}
	if g.AllocsPresent {
		columnNames = append(columnNames, "allocs")
	}
	table := &Table{Cells: [][]string{columnNames}}

	var underlines []string
	for _, name := range columnNames {
		underlines = append(underlines, strings.Repeat("-", len(name)))
	}
	table.Cells = append(table.Cells, underlines)
	timeFormatFunc := g.TimeFormatFunc()

	for _, line := range g.Lines {
		row := []string{line.Name, FormatIterations(line.N), timeFormatFunc(line.NsOp)}
		if g.MegaBytesPresent {
			row = append(row, FormatMegaBytesPerSecond(line))
		}
		if g.BytesAllocPresent {
			row = append(row, FormatBytesAllocPerOp(line))
		}
		if g.AllocsPresent {
			row = append(row, FormatAllocsPerOp(line))
		}
		table.Cells = append(table.Cells, row)
	}
	for i := range columnNames {
		maxLength := 0
		for _, row := range table.Cells {
			if len(row[i]) > maxLength {
				maxLength = len(row[i])
			}
		}
		table.MaxLengths = append(table.MaxLengths, maxLength)
	}
	var buf bytes.Buffer
	for _, row := range table.Cells {
		for i, cell := range row {
			var format string
			switch i {
			case 0:
				format = "%%-%ds   "
			case len(row) - 1:
				format = "%%%ds"
			default:
				format = "%%%ds   "
			}
			fmt.Fprintf(&buf, fmt.Sprintf(format, table.MaxLengths[i]), cell)
		}
		fmt.Fprint(&buf, "\n")
	}
	return buf.String()
}

func FormatIterations(iter int) string {
	return strconv.FormatInt(int64(iter), 10)
}

func (g *BenchOutputGroup) TimeFormatFunc() func(float64) string {
	// Find the smallest time
	smallest := g.Lines[0].NsOp
	for _, line := range g.Lines[1:] {
		if line.NsOp < smallest {
			smallest = line.NsOp
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

func FormatMegaBytesPerSecond(l *benchcmp.Bench) string {
	if (l.Measured & benchcmp.MbS) == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f MB/s", l.MbS)
}

func FormatBytesAllocPerOp(l *benchcmp.Bench) string {
	if (l.Measured & benchcmp.BOp) == 0 {
		return ""
	}
	return fmt.Sprintf("%d B/op", l.BOp)
}

func FormatAllocsPerOp(l *benchcmp.Bench) string {
	if (l.Measured & benchcmp.AllocsOp) == 0 {
		return ""
	}
	return fmt.Sprintf("%d allocs/op", l.AllocsOp)
}

func (g *BenchOutputGroup) AddLine(line *benchcmp.Bench) {
	g.Lines = append(g.Lines, line)
	if (line.Measured & benchcmp.MbS) > 0 {
		g.MegaBytesPresent = true
	}
	if (line.Measured & benchcmp.BOp) > 0 {
		g.BytesAllocPresent = true
	}
	if (line.Measured & benchcmp.AllocsOp) > 0 {
		g.AllocsPresent = true
	}
}

var (
	benchLineMatcher = regexp.MustCompile(`^Benchmark.*\t.*\d+`)
	okLineMatcher    = regexp.MustCompile(`^ok\s`)
	notBenchLineErr  = errors.New("Not a bench line")
)

func ParseLine(line string) (*benchcmp.Bench, error) {
	if !benchLineMatcher.MatchString(line) {
		return nil, notBenchLineErr
	}
	fields := strings.Split(line, "\t")
	if len(fields) < 3 {
		return nil, notBenchLineErr
	}

	return benchcmp.ParseLine(line)
}

func main() {
	flag.Parse()
	currentBenchmark := &BenchOutputGroup{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		line, err := ParseLine(text)
		switch err {
		case notBenchLineErr:
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
