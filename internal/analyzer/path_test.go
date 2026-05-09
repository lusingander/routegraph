package analyzer

import "testing"

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name string
		base string
		part string
		want string
	}{
		{name: "absolute child", base: "/api", part: "/users", want: "/api/users"},
		{name: "trailing slash", base: "/api/", part: "/users", want: "/api/users"},
		{name: "relative child", base: "/api", part: "users", want: "/api/users"},
		{name: "root parent", base: "/", part: "/users", want: "/users"},
		{name: "unknown parent", base: "<unknown>", part: "/users", want: "<unknown>/users"},
		{name: "unknown child", base: "/api", part: "<unknown>", want: "/api/<unknown>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPath(KnownPath(tt.base), KnownPath(tt.part))
			if got.Value != tt.want {
				t.Fatalf("JoinPath(%q, %q) = %q, want %q", tt.base, tt.part, got.Value, tt.want)
			}
		})
	}
}
