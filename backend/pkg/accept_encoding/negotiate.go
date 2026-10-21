// Package acceptencoding парсит заголовок Accept-Encoding (RFC 9110, 12.5.3)
// и возвращает предпочтительный encoding из поддерживаемых сервером (br, gzip).
//
// Сервер отдаёт заранее сжатые варианты файлов (.br/.gz, их делает сборка
// образа), поэтому нужно только выбрать, какой из них клиент понимает.
package accept_encoding

import (
	"strconv"
	"strings"
)

// Negotiate возвращает лучший поддерживаемый encoding из заголовка
// Accept-Encoding: "br", "gzip" или "" (сжатие не применяется).
//
// Учитываются веса (q=) и запреты (q=0): кодирование, которое клиент запретил,
// не предлагается даже через «*».
func Negotiate(header string) string {
	excluded := map[string]bool{}
	best, bestQ := "", 0.0

	for _, field := range strings.Split(header, ",") {
		name, q := parseField(field)
		if name == "" {
			continue
		}
		if q <= 0 {
			excluded[name] = true
			continue
		}
		switch name {
		case "br", "gzip":
		case "*":
			// «*» — любое не названное явно кодирование; при равном весе
			// предпочитаем br (он меньше gzip).
			name = firstAllowed(excluded)
			if name == "" {
				continue
			}
		default:
			continue
		}
		if q > bestQ || (q == bestQ && name == "br" && best != "br") {
			best, bestQ = name, q
		}
	}
	return best
}

// firstAllowed — кодирование по умолчанию для «*»: br меньше gzip, а
// запрещённые клиентом (q=0) не предлагаются.
func firstAllowed(excluded map[string]bool) string {
	for _, enc := range []string{"br", "gzip"} {
		if !excluded[enc] {
			return enc
		}
	}
	return ""
}

// parseField разбирает поле заголовка «name;q=0.5»: имя кодирования и его вес.
// Имя приводится к нижнему регистру (кодирования нечувствительны к регистру),
// отсутствие q — вес 1.
func parseField(field string) (name string, q float64) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", 0
	}
	name, params, _ := strings.Cut(field, ";")
	name = strings.ToLower(strings.TrimSpace(name))
	q = 1
	for _, p := range strings.Split(params, ";") {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(p, "q=") {
			continue
		}
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(p[2:]), 64); err == nil {
			q = parsed
		}
	}
	return name, q
}
