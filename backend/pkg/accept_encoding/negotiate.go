// Package acceptencoding парсит заголовок Accept-Encoding (RFC 7231, Section 5.3.4)
// и возвращает предпочтительный encoding из поддерживаемых сервером (br, gzip).
package accept_encoding

import (
	"strconv"
	"strings"
)

// Negotiate возвращает лучший поддерживаемый encoding из заголовка Accept-Encoding.
// Возвращает "" (без компрессии), если клиент не принимает br или gzip.
func Negotiate(header string) string {
	if header == "" {
		return ""
	}

	type item struct {
		name    string
		quality float64
	}
	var preferred []item
	excluded := map[string]bool{}

	for _, field := range strings.Split(header, ",") {
		enc, q := parseField(field)
		if enc == "" {
			continue
		}
		if q == 0 {
			if enc == "identity" || enc == "*" {
				return ""
			}
			excluded[enc] = true
			continue
		}
		if enc == "br" || enc == "gzip" || enc == "*" {
			preferred = append(preferred, item{enc, q})
		}
	}

	bestQ := 0.0
	best := ""
	for _, it := range preferred {
		enc := resolve(it.name, excluded)
		if enc == "" {
			continue
		}
		if it.quality > bestQ || (it.quality == bestQ && enc == "br" && best != "br") {
			best, bestQ = enc, it.quality
		}
	}
	return best
}

func parseField(field string) (name string, quality float64) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", 0
	}
	quality = 1.0

	idx := strings.IndexByte(field, ';')
	if idx == -1 {
		return field, quality
	}

	name = strings.TrimSpace(field[:idx])
	rest := field[idx+1:]
	if qi := strings.Index(rest, "q="); qi != -1 {
		qStr := strings.TrimSpace(rest[qi+2:])
		if end := strings.IndexByte(qStr, ';'); end != -1 {
			qStr = strings.TrimSpace(qStr[:end])
		}
		if parsed, err := strconv.ParseFloat(qStr, 64); err == nil {
			quality = parsed
		}
	}
	return name, quality
}

func resolve(name string, excluded map[string]bool) string {
	if name == "*" {
		if !excluded["gzip"] {
			return "gzip"
		}
		if !excluded["br"] {
			return "br"
		}
		return ""
	}
	if excluded[name] {
		return ""
	}
	return name
}
