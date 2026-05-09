package printer

import (
	"bytes"
	"testing"

	"github.com/lusingander/routegraph"
)

func TestPrintJSON(t *testing.T) {
	routes := []routegraph.Route{
		{
			Framework: "echo",
			Method:    "GET",
			Path:      "/api/v1/users",
			Handler:   "listUsers",
			File:      "internal/routes/user.go",
			Line:      24,
			Known:     true,
		},
	}

	var out bytes.Buffer
	if err := PrintJSON(&out, routes); err != nil {
		t.Fatal(err)
	}

	want := "[\n" +
		"  {\n" +
		"    \"framework\": \"echo\",\n" +
		"    \"method\": \"GET\",\n" +
		"    \"path\": \"/api/v1/users\",\n" +
		"    \"handler\": \"listUsers\",\n" +
		"    \"file\": \"internal/routes/user.go\",\n" +
		"    \"line\": 24,\n" +
		"    \"known\": true,\n" +
		"    \"warnings\": null\n" +
		"  }\n" +
		"]\n"
	if out.String() != want {
		t.Fatalf("PrintJSON() =\n%q\nwant\n%q", out.String(), want)
	}
}
