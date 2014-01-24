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
)

var noPassthrough = flag.Bool("no-passthrough", false, "Don't print non-benchmark lines")

type BenchOutput struct {
	Name             string
	Iterations       int64
	TimePerIteration time.Duration
	// Optional fields
	BytesPerSecond  *float64
	BytesAllocPerOp *int64
	AllocsPerOp     *int64
}

type BenchOutputGroup struct {
	Lines []*BenchOutput
	// Columns which are in use
	BytesPresent      bool
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
	if g.BytesPresent {
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
		row := []string{line.Name, FormatIterations(line.Iterations), timeFormatFunc(line.TimePerIteration)}
		if g.BytesPresent {
			row = append(row, FormatBytesPerSecond(line.BytesPerSecond))
		}
		if g.BytesAllocPresent {
			row = append(row, FormatBytesAllocPerOp(line.BytesAllocPerOp))
		}
		if g.AllocsPresent {
			row = append(row, FormatAllocsPerOp(line.AllocsPerOp))
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

func FormatIterations(iter int64) string {
	return strconv.FormatInt(iter, 10)
}

func (g *BenchOutputGroup) TimeFormatFunc() func(time.Duration) string {
	// Find the smallest time
	smallest := g.Lines[0].TimePerIteration
	for _, line := range g.Lines[1:] {
		if line.TimePerIteration < smallest {
			smallest = line.TimePerIteration
		}
	}
	switch {
	case smallest < 10000*time.Nanosecond:
		return func(d time.Duration) string {
			return fmt.Sprintf("%d ns/op", d.Nanoseconds())
		}
	case smallest < time.Millisecond:
		return func(d time.Duration) string {
			return fmt.Sprintf("%.2f μs/op", float64(d.Nanoseconds())/1000)
		}
	case smallest < 10*time.Second:
		return func(d time.Duration) string {
			return fmt.Sprintf("%.2f ms/op", d.Seconds()*1000)
		}
	default:
		return func(d time.Duration) string {
			return fmt.Sprintf("%.2f s/op", d.Seconds())
		}
	}
}

func FormatBytesPerSecond(b *float64) string {
	if b == nil {
		return ""
	}
	return fmt.Sprintf("%.2f MB/s", *b/1e6)
}

func FormatBytesAllocPerOp(b *int64) string {
	if b == nil {
		return ""
	}
	return fmt.Sprintf("%d B/op", *b)
}

func FormatAllocsPerOp(a *int64) string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%d allocs/op", *a)
}

func (g *BenchOutputGroup) AddLine(line *BenchOutput) {
	g.Lines = append(g.Lines, line)
	if line.BytesPerSecond != nil {
		g.BytesPresent = true
	}
	if line.BytesAllocPerOp != nil {
		g.BytesAllocPresent = true
	}
	if line.AllocsPerOp != nil {
		g.AllocsPresent = true
	}
}

var (
	benchLineMatcher  = regexp.MustCompile(`^Benchmark.*\t.*\d+`)
	okLineMatcher     = regexp.MustCompile(`^ok\s`)
	notBenchLineErr   = errors.New("Not a bench line")
	benchLineParseErr = errors.New("Unable to parse benchmark output line")
)

func ParseLine(line string) (*BenchOutput, error) {
	if !benchLineMatcher.MatchString(line) {
		return nil, notBenchLineErr
	}
	fields := strings.Split(line, "\t")
	if len(fields) < 3 {
		return nil, notBenchLineErr
	}
	var err error
	output := &BenchOutput{Name: strings.TrimSpace(fields[0])}
	output.Iterations, err = strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
	if err != nil {
		return nil, notBenchLineErr
	}
	for _, field := range fields[2:] {
		parts := strings.Split(strings.TrimSpace(field), " ")
		if len(parts) != 2 {
			return nil, benchLineParseErr
		}
		if parts[1] == "MB/s" {
			value, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return nil, benchLineParseErr
			}
			value *= 1e6
			output.BytesPerSecond = &value
			continue
		}
		value, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, benchLineParseErr
		}
		switch parts[1] {
		case "ns/op":
			output.TimePerIteration = time.Duration(value)
		case "B/op":
			output.BytesAllocPerOp = &value
		case "allocs/op":
			output.AllocsPerOp = &value
		default:
			return nil, benchLineParseErr
		}
	}
	return output, nil
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
