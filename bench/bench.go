// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"fmt"
	"strconv"
	"strings"
)

// Flags used by Bench.Measured to indicate
// which measurements a Bench contains.
const (
	NsOp = 1 << iota
	MbS
	BOp
	AllocsOp
)

// Bench is one run of a single benchmark.
type Bench struct {
	Name     string  // benchmark name
	N        int     // number of iterations
	NsOp     float64 // nanoseconds per iteration
	MbS      float64 // MB processed per second
	BOp      uint64  // bytes allocated per iteration
	AllocsOp uint64  // allocs per iteration
	Measured int     // which measurements were recorded
	ord      int     // ordinal position within a benchmark run, used for sorting
}

// ParseLine extracts a Bench from a single line of testing.B output.
func ParseLine(line string) (*Bench, error) {
	fields := strings.Fields(line)

	// Two required, positional fields: Name and iterations.
	if len(fields) < 2 {
		return nil, fmt.Errorf("two fields required, have %d", len(fields))
	}
	if !strings.HasPrefix(fields[0], "Benchmark") {
		return nil, fmt.Errorf(`first field does not start with "Benchmark`)
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, err
	}
	b := &Bench{Name: fields[0], N: n}

	// Parse any remaining pairs of fields; we've parsed one pair already.
	for i := 1; i < len(fields)/2; i++ {
		b.parseMeasurement(fields[i*2], fields[i*2+1])
	}
	return b, nil
}

func (b *Bench) parseMeasurement(quant string, unit string) {
	switch unit {
	case "ns/op":
		if f, err := strconv.ParseFloat(quant, 64); err == nil {
			b.NsOp = f
			b.Measured |= NsOp
		}
	case "MB/s":
		if f, err := strconv.ParseFloat(quant, 64); err == nil {
			b.MbS = f
			b.Measured |= MbS
		}
	case "B/op":
		if i, err := strconv.ParseUint(quant, 10, 64); err == nil {
			b.BOp = i
			b.Measured |= BOp
		}
	case "allocs/op":
		if i, err := strconv.ParseUint(quant, 10, 64); err == nil {
			b.AllocsOp = i
			b.Measured |= AllocsOp
		}
	}
}
