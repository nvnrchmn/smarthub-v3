package usecase

import "testing"

func TestParsePagination(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		wantHal, wantPer, wantOff int
	}{
		{"kosong", map[string]string{}, 1, 50, 0},
		{"hal2", map[string]string{"hal": "2"}, 2, 50, 50},
		{"hal0→1", map[string]string{"hal": "0"}, 1, 50, 0},
		{"negatif", map[string]string{"hal": "-3"}, 1, 50, 0},
		{"max10", map[string]string{"max": "10"}, 1, 10, 0},
		{"max300→200", map[string]string{"max": "300"}, 1, 200, 0},
		{"bukan-angka", map[string]string{"hal": "abc"}, 1, 50, 0},
		{"lengkap", map[string]string{"hal": "3", "max": "25"}, 3, 25, 50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := ParsePagination(c.in)
			if p.Page != c.wantHal || p.Per != c.wantPer || p.Offset() != c.wantOff {
				t.Fatalf("%s: dapat=%+v", c.name, p)
			}
		})
	}
}
