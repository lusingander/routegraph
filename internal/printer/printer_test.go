package printer

import (
	"bytes"
	"testing"

	"github.com/lusingander/routegraph"
)

func TestPrint(t *testing.T) {
	routes := []routegraph.Route{
		{
			Method:  "GET",
			Path:    "/api/v1/users",
			Handler: "listUsers",
			File:    "internal/routes/user.go",
			Line:    24,
		},
		{
			Method:  "POST",
			Path:    "<unknown>/users",
			Handler: "createUser",
			File:    "internal/routes/user.go",
			Line:    25,
		},
	}

	var out bytes.Buffer
	if err := Print(&out, routes); err != nil {
		t.Fatal(err)
	}

	want := "GET   /api/v1/users    listUsers   internal/routes/user.go:24\n" +
		"POST  <unknown>/users  createUser  internal/routes/user.go:25\n"
	if out.String() != want {
		t.Fatalf("Print() =\n%q\nwant\n%q", out.String(), want)
	}
}
