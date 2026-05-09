package analyzer

import "strings"

func KnownPath(value string) PathExpr {
	return PathExpr{Value: value, Known: true}
}

func UnknownPath(reason string) PathExpr {
	return PathExpr{Value: "<unknown>", Known: false, Reason: reason}
}

func JoinPath(base, part PathExpr) PathExpr {
	value := joinPathValue(base.Value, part.Value)
	known := base.Known && part.Known
	reason := ""
	if !known {
		reason = base.Reason
		if reason == "" {
			reason = part.Reason
		}
	}
	return PathExpr{Value: value, Known: known, Reason: reason}
}

func joinPathValue(base, part string) string {
	if base == "" || base == "/" {
		return ensureLeadingSlash(part)
	}
	if part == "" || part == "/" {
		return ensureLeadingSlash(strings.TrimRight(base, "/"))
	}
	return ensureLeadingSlash(strings.TrimRight(base, "/") + "/" + strings.TrimLeft(part, "/"))
}

func ensureLeadingSlash(value string) string {
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "<unknown>") {
		return value
	}
	return "/" + value
}
