// Package hub — klien tipis untuk Logikraf Payment Hub (QRIS dinamis).
//
// Hub tidak mengirim webhook untuk QRIS, jadi penautan "QRIS lunas -> invoice
// lunas" dilakukan di sisi Smarthub: status referensi diperiksa ke hub, lalu
// invoice ditandai lunas secara idempoten.
package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrQRISTidakSah — hub membalas QR pendek (placeholder) sehingga QRIS tidak bisa dipakai.
var ErrQRISTidakSah = errors.New("qr_string dari hub tidak sah")

// ErrGatewayBermasalah — hub membalas galat server (akun pembayaran belum siap, dsb).
var ErrGatewayBermasalah = errors.New("gateway pembayaran sedang bermasalah")

// ErrSudahDibayar — hub menolak karena pembayaran referensi ini sudah lunas (409).
var ErrSudahDibayar = errors.New("pembayaran sudah lunas di gateway")

type Client struct {
	base   string
	key    string
	prefix string
	http   *http.Client
}

func New() *Client {
	return &Client{
		base:   strings.TrimRight(strings.TrimSpace(os.Getenv("HUB_BASE_URL")), "/"),
		key:    strings.TrimSpace(os.Getenv("HUB_INTERNAL_KEY")),
		prefix: strings.TrimSpace(os.Getenv("HUB_EXT_PREFIX")),
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) Enabled() bool { return c.base != "" && c.key != "" }

func (c *Client) ExternalID(invoiceNumber string) string {
	return c.prefix + invoiceNumber
}

// ResponsCreate — jawaban hub saat QRIS dibuat.
type ResponsCreate struct {
	ReferenceID string  `json:"reference_id"`
	ExternalID  string  `json:"external_id"`
	QRString    string  `json:"qr_string"`
	Amount      float64 `json:"amount"`
	ExpiresAt   string  `json:"expires_at"`
	Status      string  `json:"status"`
}

// ResponsStatus — jawaban hub saat status diperiksa (endpoint publik).
type ResponsStatus struct {
	ReferenceID string  `json:"reference_id"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
	PaidAt      string  `json:"paid_at"`
}

// CreateQRIS — buat QRIS dinamis untuk sebuah invoice.
func (c *Client) CreateQRIS(ctx context.Context, externalID string, amount float64, description string) (*ResponsCreate, error) {
	if !c.Enabled() {
		return nil, errors.New("payment hub belum dikonfigurasi")
	}
	body, _ := json.Marshal(map[string]any{
		"external_id": externalID, "amount": amount, "description": description,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/client-store-qris", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w (%d): %s", ErrGatewayBermasalah, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("%w: %s", ErrSudahDibayar, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hub menolak (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out ResponsCreate
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("respons hub tidak terbaca: %w", err)
	}
	// Penjaga penting: QRIS gagal biasanya datang sebagai string pendek (placeholder).
	if len(strings.TrimSpace(out.QRString)) < 150 {
		return nil, fmt.Errorf("%w: %d karakter", ErrQRISTidakSah, len(out.QRString))
	}
	return &out, nil
}

// Status — periksa status QRIS berdasarkan reference_id.
func (c *Client) Status(ctx context.Context, referenceID string) (*ResponsStatus, error) {
	if !c.Enabled() {
		return nil, errors.New("payment hub belum dikonfigurasi")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.base+"/api/payment/qris/"+referenceID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hub menolak status (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out ResponsStatus
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("respons status hub tidak terbaca: %w", err)
	}
	return &out, nil
}

// SudahLunas — hub memakai beberapa penamaan status; perlakukan set ini sebagai lunas.
func (r *ResponsStatus) SudahLunas() bool {
	switch strings.ToLower(strings.TrimSpace(r.Status)) {
	case "paid", "settled", "succeeded", "success":
		return true
	}
	return false
}
