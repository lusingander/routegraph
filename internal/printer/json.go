package printer

import (
	"encoding/json"
	"io"

	"github.com/lusingander/routegraph"
)

func PrintJSON(w io.Writer, routes []routegraph.Route) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(routes)
}
