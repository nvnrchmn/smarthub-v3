package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Sender — kanal pengiriman OTP/undangan lewat gateway WhatsApp (GoWA) yang
// sudah berjalan di server. Bila WA_BASE_URL kosong, kode hanya dicatat ke log
// supaya alur tetap bisa diuji tanpa mengirim apa pun ke pihak luar.
type Sender struct {
	waBase   string
	waAuth   string
	waDevice string
	devMode  bool
	client   *http.Client
}

func New() *Sender {
	base := strings.TrimSpace(os.Getenv("WA_BASE_URL"))
	return &Sender{
		waBase:   strings.TrimRight(base, "/"),
		waAuth:   strings.TrimSpace(os.Getenv("WA_BASIC_AUTH")),
		waDevice: strings.TrimSpace(os.Getenv("WA_DEVICE_ID")),
		devMode:  base == "",
		client:   &http.Client{Timeout: 20 * time.Second},
	}
}

// KanalAktif — dipakai API untuk memberi tahu pengurus apakah pesan benar-benar terkirim.
func (s *Sender) KanalAktif() bool { return !s.devMode }

// SendOTP — kirim kode aktivasi ke nomor warga.
func (s *Sender) SendOTP(phone, code, tenantName string) error {
	msg := "Kode aktivasi Smarthub" + tenantSuffix(tenantName) + ": " + code +
		" (berlaku 10 menit). Jangan bagikan kode ini kepada siapa pun."
	return s.send(phone, msg)
}

// SendTagihan — pemberitahuan tagihan iuran bulanan.
func (s *Sender) SendTagihan(phone, pesan, tenantName string) error {
	return s.send(phone, pesan+tenantSuffix(tenantName))
}

// SendLupaSandi — kirim kode reset password ke WhatsApp pengguna.
func (s *Sender) SendLupaSandi(phone, code, tenantName string) error {
	msg := "Kode reset password Smarthub" + tenantSuffix(tenantName) + ": " + code +
		" (berlaku 1 jam). Jangan bagikan kode ini kepada siapa pun."
	return s.send(phone, msg)
}

// SendInviteLink — tautan aktivasi (berlaku 7 hari).
func (s *Sender) SendInviteLink(phone, link, tenantName string) error {
	msg := "Undangan Smarthub" + tenantSuffix(tenantName) + ": aktivasi akun Anda di " + link +
		" (tautan berlaku 7 hari)."
	return s.send(phone, msg)
}

// SendPengingatTunggakan — kirim pengingat tagihan yang belum dibayar.
func (s *Sender) SendPengingatTunggakan(phone, pesan, tenantName string) error {
	msg := "Pengingat Smarthub" + tenantSuffix(tenantName) + ": " + pesan +
		". Harap segera lakukan pembayaran."
	return s.send(phone, msg)
}

func tenantSuffix(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return " — " + name
}

var nonDigit = regexp.MustCompile(`\D`)

// NormalisasiNomor — nomor Indonesia ditulis warga dalam banyak bentuk
// (0812…, +62 812-…); gateway memakai format 62812….
func NormalisasiNomor(raw string) string {
	n := nonDigit.ReplaceAllString(strings.TrimSpace(raw), "")
	switch {
	case n == "":
		return ""
	case strings.HasPrefix(n, "62"):
		return n
	case strings.HasPrefix(n, "0"):
		return "62" + strings.TrimLeft(n, "0")
	case strings.HasPrefix(n, "8"):
		return "62" + n
	}
	return n
}

type waResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Sender) send(phone, message string) error {
	nomor := NormalisasiNomor(phone)
	if s.devMode {
		// Tanpa kanal WhatsApp: kode/tautan hanya masuk log server (bukan ke pihak luar).
		log.Printf("[notify:dev] ke %s: %s", maskPhone(nomor), message)
		return nil
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		lastErr = s.kirimSekali(nomor, message)
		if lastErr == nil {
			log.Printf("[notify] WhatsApp terkirim ke %s (perangkat %s)", maskPhone(nomor), s.waDevice)
			return nil
		}
		if attempt < 3 {
			// Hanya kegagalan sementara yang layak diulang.
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			log.Printf("[notify] percobaan %d gagal (%v), ulangi", attempt, lastErr)
		}
	}
	return fmt.Errorf("gagal mengirim WhatsApp ke %s: %w", maskPhone(nomor), lastErr)
}

func (s *Sender) kirimSekali(nomor, message string) error {
	body, _ := json.Marshal(map[string]string{
		"phone": nomor, "message": message, "device_id": s.waDevice,
	})
	req, err := http.NewRequest(http.MethodPost, s.waBase+"/send/message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.waAuth != "" {
		u, p := splitAuth(s.waAuth)
		req.SetBasicAuth(u, p)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 500 {
		return fmt.Errorf("gateway bermasalah (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gateway menolak (HTTP %d): %s", resp.StatusCode, potong(string(raw), 160))
	}
	// GoWA menjawab 200 dengan code SUCCESS bila pesan diterima; selain itu gagal.
	var r waResponse
	if err := json.Unmarshal(raw, &r); err == nil && r.Code != "" && !strings.EqualFold(r.Code, "SUCCESS") {
		return fmt.Errorf("gateway menolak: %s %s", r.Code, potong(r.Message, 120))
	}
	return nil
}

func potong(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func splitAuth(v string) (string, string) {
	if i := strings.Index(v, ":"); i > 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func maskPhone(p string) string {
	if len(p) <= 4 {
		return "***"
	}
	return p[:4] + "****" + p[len(p)-2:]
}
