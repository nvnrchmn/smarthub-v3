package usecase

import (
	"strconv"
)

// Pagination mengurai & membatasi parameter paginasi dari URL.
//
// Parameter: `hal` (1-based) dan `max` per halaman.
// Pembatasan: hal minimal 1, max 1-200. Nilai di luar batas dibulatkan ke
// amanannya — tidak ada error karena parameter pagination.
type Pagination struct {
	Page int // 1-based
	Per  int // jumlah per halaman
}

// ParsePagination mengembalikan Pagination dari query string URL.
// amanParsing=false karena fungsi ini tidak sensitif terhadap kesalahan
// parsing: input tidak valid cukup dibakukan ke default.
func ParsePagination(qs map[string]string) Pagination {
	p := Pagination{Page: 1, Per: 50}
	if h, err := strconv.Atoi(qs["hal"]); err == nil && h >= 1 {
		p.Page = h
	}
	if m, err := strconv.Atoi(qs["max"]); err == nil {
		switch {
		case m < 1:
			p.Per = 50
		case m > 200:
			p.Per = 200
		default:
			p.Per = m
		}
	}
	return p
}

// Offset mengembalikan OFFSET SQL dari halaman & jumlah per halaman.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Per
}

// Paginated adalah pembungkus response yang menyertakan info pagination.
type Paginated[T any] struct {
	Items    T      `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PerPage  int    `json:"per_page"`
	LastPage int    `json:"last_page"`
}
