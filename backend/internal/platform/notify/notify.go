package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Sender — kanal pengiriman OTP/undangan. Bila WhatsApp belum dikonfigurasi,
// kode dicatat ke log server supaya alur tetap bisa diuji tanpa mengirim apa pun.
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
		waBase:   base,
		waAuth:   strings.TrimSpace(os.Getenv("WA_BASIC_AUTH")),
		waDevice: strings.TrimSpace(os.Getenv("WA_DEVICE_ID")),
		devMode:  base == "",
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// SendOTP — kirim kode ke nomor warga.
func (s *Sender) SendOTP(phone, code, tenantName string) error {
	msg := "Kode aktivasi Smarthub" + tenantSuffix(tenantName) + ": " + code +
		" (berlaku 10 menit). Jangan bagikan kode ini kepada siapa pun."
	return s.send(phone, msg)
}

// SendInviteLink — tautan aktivasi (berlaku 7 hari).
func (s *Sender) SendInviteLink(phone, link, tenantName string) error {
	msg := "Undangan Smarthub" + tenantSuffix(tenantName) + ": aktivasi akun Anda di " + link +
		" (tautan berlaku 7 hari)."
	return s.send(phone, msg)
}

func tenantSuffix(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return " — " + name
}

func (s *Sender) send(phone, message string) error {
	if s.devMode {
		// Tanpa kanal WhatsApp: kode/tauatan hanya masuk log server (bukan ke pihak luar).
		log.Printf("[notify:dev] ke %s: %s", maskPhone(phone), message)
		return nil
	}
	body, _ := json.Marshal(map[string]string{
		"phone": phone, "message": message, "device_id": s.waDevice,
	})
	req, err := http.NewRequest(http.MethodPost, s.waBase+"/send/message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.waAuth != "" {
		req.SetBasicAuth(splitAuth(s.waAuth))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[notify] gateway WhatsApp menjawab %d untuk %s", resp.StatusCode, maskPhone(phone))
	}
	return nil
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
