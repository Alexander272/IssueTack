package accept_encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNegotiate(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"заголовка нет", "", ""},
		{"только br", "br", "br"},
		{"только gzip", "gzip", "gzip"},
		{"оба равны — br", "gzip, br", "br"},
		{"br важнее по весу", "gzip;q=0.5, br;q=0.9", "br"},
		{"gzip важнее по весу", "br;q=0.5, gzip;q=0.9", "gzip"},
		{"br запрещён", "br;q=0, gzip", "gzip"},
		{"gzip запрещён", "gzip;q=0, br", "br"},
		{"оба запрещены", "br;q=0, gzip;q=0", ""},
		{"звезда", "*", "br"},
		{"звезда при запрете br", "br;q=0, *", "gzip"},
		{"неизвестное кодирование", "deflate", ""},
		{"запрет gzip", "gzip;q=0", ""},
		{"регистр не важен", "BR", "br"},
		{"дополнительные параметры", "br;foo=1;q=0.8", "br"},
		{"пустые поля", ",,br,,", "br"},
		{"identity ничего не меняет", "identity, gzip", "gzip"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Negotiate(tt.header))
		})
	}
}
