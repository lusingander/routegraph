package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsRoutes(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"../../testdata/basic"}, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"GET   /api/v1/users        listUsers",
		"POST  /api/v1/users        createUser",
		"GET   /api/v1/admin/stats  h.Stats",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output does not contain %q:\n%s", want, got)
		}
	}
}

func TestRunAcceptsRecursivePackagePattern(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"../../testdata/basic/..."}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "/api/v1/users") {
		t.Fatalf("route not printed:\n%s", out.String())
	}
}

func TestRunPrintsJSON(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"--json", "../../testdata/basic"}, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		`"framework": "echo"`,
		`"method": "GET"`,
		`"path": "/api/v1/users"`,
		`"handler": "listUsers"`,
		`"known": true`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output does not contain %q:\n%s", want, got)
		}
	}
}
