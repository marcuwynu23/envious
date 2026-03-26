package view

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

func (t Table) Render(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(t.Headers) > 0 {
		fmt.Fprintln(tw, strings.Join(t.Headers, "\t"))
		sep := make([]string, 0, len(t.Headers))
		for _, h := range t.Headers {
			n := len(h)
			if n < 2 {
				n = 2
			}
			sep = append(sep, strings.Repeat("─", n))
		}
		fmt.Fprintln(tw, strings.Join(sep, "\t"))
	}
	for _, r := range t.Rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	_ = tw.Flush()
}

