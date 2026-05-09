package printer

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/lusingander/routegraph"
)

func Print(w io.Writer, routes []routegraph.Route) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, route := range routes {
		location := route.File
		if route.Line > 0 {
			location = fmt.Sprintf("%s:%d", route.File, route.Line)
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", route.Method, route.Path, route.Handler, location); err != nil {
			return err
		}
	}
	return tw.Flush()
}
