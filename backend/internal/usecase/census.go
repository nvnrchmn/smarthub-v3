package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/storage"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

var (
	ErrForbidden = errors.New("tidak berhak")
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrBadInput  = errors.New("data tidak valid")
)

type Census struct {
	Store   *postgres.Store
	Storage *storage.Store
}

func resourceOf(p *domain.ResidentProfile) domain.ResourceContext {
	return domain.ResourceContext{
		TenantID:     "", // diisi pemanggil
		HouseUnitID:  p.HouseUnitID,
		FamilyCardID: p.FamilyCardID,
		IsSensitive:  true,
	}
}

// SubmitMyProfile — warga mengisi/memperbarui data sensusnya.
func (c *Census) SubmitMyProfile(ctx context.Context, sub *domain.SubjectContext, in postgres.ProfileInput) (*domain.ResidentProfile, error) {
	if strings.TrimSpace(in.FullName) == "" {
		return nil, fmt.Errorf("%w: nama lengkap wajib", ErrBadInput)
	}
	if in.NIK != "" && (len(in.NIK) != 16 || !allDigits(in.NIK)) {
		return nil, fmt.Errorf("%w: NIK harus 16 digit angka", ErrBadInput)
	}
	if in.KKNumber != "" && (len(in.KKNumber) != 16 || !allDigits(in.KKNumber)) {
		return nil, fmt.Errorf("%w: nomor KK harus 16 digit angka", ErrBadInput)
	}
	if in.FamilyRole == "" {
		in.FamilyRole = domain.FamilyRoleOther
	}
	p, err := c.Store.UpsertMyProfile(ctx, sub.TenantID, sub.AccountID, in)
	if err != nil {
		return nil, err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "CENSUS_SUBMIT", "resident_profile", p.ID,
		map[string]any{"nik_diisi": in.NIK != "", "kk_diisi": in.KKNumber != ""})
	return p, nil
}

// MyProfile — profil sensus milik akun yang masuk.
func (c *Census) MyProfile(ctx context.Context, sub *domain.SubjectContext) (*domain.ResidentProfile, error) {
	p, err := c.Store.GetProfileByAccount(ctx, sub.TenantID, sub.AccountID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

// List — daftar sensus untuk pengurus (data tetap tersamar).
func (c *Census) List(ctx context.Context, sub *domain.SubjectContext, status string, limit, offset int) ([]domain.ResidentProfile, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	return c.Store.ListProfiles(ctx, sub.TenantID, status, limit, offset)
}

func (c *Census) Count(ctx context.Context, sub *domain.SubjectContext, status string) (int, error) {
	if !c.isStaff(sub) {
		return 0, ErrForbidden
	}
	return c.Store.CountProfiles(ctx, sub.TenantID, status)
}

// Detail — satu profil. Warga hanya boleh membuka miliknya sendiri.
func (c *Census) Detail(ctx context.Context, sub *domain.SubjectContext, id string) (*domain.ResidentProfile, error) {
	p, err := c.Store.GetProfileByID(ctx, sub.TenantID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	res := resourceOf(p)
	res.TenantID = sub.TenantID
	if !domain.CanAccess(*sub, res, domain.ActionViewPII) {
		return nil, ErrForbidden
	}
	return p, nil
}

// RevealPII — membuka NIK & nomor KK asli. Selalu dicatat ke audit_logs.
func (c *Census) RevealPII(ctx context.Context, sub *domain.SubjectContext, id string) (string, string, error) {
	p, err := c.Store.GetProfileByID(ctx, sub.TenantID, id)
	if err != nil {
		return "", "", err
	}
	if p == nil {
		return "", "", ErrNotFound
	}
	res := resourceOf(p)
	res.TenantID = sub.TenantID
	if !domain.CanAccess(*sub, res, domain.ActionViewPII) {
		_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "PII_ACCESS_DENIED", "resident_profile", id, nil)
		return "", "", ErrForbidden
	}
	nik, kk, err := c.Store.DecryptPII(ctx, sub.TenantID, id)
	if err != nil {
		return "", "", err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "PII_VIEW", "resident_profile", id,
		map[string]any{"peran": sub.AppRoles})
	return nik, kk, nil
}

// Verify — Sekretaris/Ketua menyetujui atau menolak kelengkapan dokumen.
func (c *Census) Verify(ctx context.Context, sub *domain.SubjectContext, id, status, reason string) error {
	if status != domain.VerifyVerified && status != domain.VerifyRejected {
		return fmt.Errorf("%w: status verifikasi tidak dikenal", ErrBadInput)
	}
	p, err := c.Store.GetProfileByID(ctx, sub.TenantID, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrNotFound
	}
	res := resourceOf(p)
	res.TenantID = sub.TenantID
	if !domain.CanAccess(*sub, res, domain.ActionVerifyPII) {
		return ErrForbidden
	}
	if status == domain.VerifyVerified && (!p.HasKTP || !p.HasKK) {
		return fmt.Errorf("%w: KTP dan KK harus diunggah sebelum disetujui", ErrBadInput)
	}
	if err := c.Store.SetVerification(ctx, sub.TenantID, id, status, sub.AccountID, reason); err != nil {
		return err
	}
	action := "CENSUS_VERIFY"
	if status == domain.VerifyRejected {
		action = "CENSUS_REJECT"
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, action, "resident_profile", id, map[string]any{"alasan": reason})
	return nil
}

// UploadDocument — menyimpan KTP/KK ke bucket privat.
//
// Keamanan: tipe konten DARI ISI BERKAS (512 byte pertama), bukan dari header
// Content-Type klien yang bisa dipalsukan. Seluruh berkas dibaca ke buffer
// dengan batas 8 MB — melebihi itu ditolak sebelum sempat diproses.
func (c *Census) UploadDocument(ctx context.Context, sub *domain.SubjectContext, id, kind string, r io.Reader, size int64, ctype string) (string, error) {
	kind = strings.ToUpper(kind)
	if kind != "KTP" && kind != "KK" {
		return "", fmt.Errorf("%w: jenis dokumen harus KTP atau KK", ErrBadInput)
	}
	if size <= 0 || size > 8<<20 {
		return "", fmt.Errorf("%w: ukuran dokumen maksimal 8 MB", ErrBadInput)
	}
	if c.Storage == nil {
		return "", fmt.Errorf("penyimpanan dokumen belum dikonfigurasi")
	}
	p, err := c.Store.GetProfileByID(ctx, sub.TenantID, id)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", ErrNotFound
	}
	res := resourceOf(p)
	res.TenantID = sub.TenantID
	if !domain.CanAccess(*sub, res, domain.ActionEditProfile) {
		return "", ErrForbidden
	}
	if kind == "KK" && p.FamilyCardID == "" {
		return "", fmt.Errorf("%w: nomor KK belum diisi pada profil", ErrBadInput)
	}

	// Baca ke buffer dengan batas 8 MB. io.LimitedReader memastikan tidak
	// lebih dari itu yang dibaca meskipun klien mengirim lebih.
	buf, err := io.ReadAll(io.LimitReader(r, 8<<20))
	if err != nil {
		return "", fmt.Errorf("%w: gagal membaca berkas", ErrBadInput)
	}
	if len(buf) == 0 {
		return "", fmt.Errorf("%w: berkas kosong", ErrBadInput)
	}

	// Deteksi tipe konten dari isi berkas, bukan dari header klien.
	// http.DetectContentType membaca 512 byte pertama dan mengembalikan MIME.
	aktual := http.DetectContentType(buf)
	switch {
	case strings.HasPrefix(aktual, "image/"):
	case aktual == "application/pdf":
	default:
		return "", fmt.Errorf("%w: dokumen harus gambar atau PDF (terdeteksi: %s)", ErrBadInput, aktual)
	}

	ext := ekstensiDariMIME(aktual)
	key := fmt.Sprintf("%s/%s/%s-%d%s", sub.TenantID, id, strings.ToLower(kind), time.Now().Unix(), ext)
	if err := c.Storage.Put(ctx, key, bytes.NewReader(buf), int64(len(buf)), aktual); err != nil {
		return "", err
	}
	if kind == "KTP" {
		err = c.Store.SetKTPPath(ctx, sub.TenantID, id, key)
	} else {
		err = c.Store.SetKKPath(ctx, sub.TenantID, p.FamilyCardID, key)
	}
	if err != nil {
		return "", err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "DOCUMENT_UPLOAD", "resident_profile", id,
		map[string]any{"jenis": kind})
	return key, nil
}

// ekstensiDariMIME mengembalikan ekstensi berkas dari tipe MIME yang dideteksi.
func ekstensiDariMIME(mime string) string {
	switch {
	case strings.Contains(mime, "jpeg"):
		return ".jpg"
	case strings.Contains(mime, "png"):
		return ".png"
	case strings.Contains(mime, "gif"):
		return ".gif"
	case strings.Contains(mime, "webp"):
		return ".webp"
	case strings.Contains(mime, "pdf"):
		return ".pdf"
	}
	return ".bin"
}

// DownloadDocument — pengurus (atau pemiliknya) mengambil dokumen; dicatat.
func (c *Census) DownloadDocument(ctx context.Context, sub *domain.SubjectContext, id, kind string) (io.ReadCloser, string, int64, error) {
	if c.Storage == nil {
		return nil, "", 0, fmt.Errorf("penyimpanan dokumen belum dikonfigurasi")
	}
	p, err := c.Store.GetProfileByID(ctx, sub.TenantID, id)
	if err != nil {
		return nil, "", 0, err
	}
	if p == nil {
		return nil, "", 0, ErrNotFound
	}
	res := resourceOf(p)
	res.TenantID = sub.TenantID
	if !domain.CanAccess(*sub, res, domain.ActionViewPII) {
		return nil, "", 0, ErrForbidden
	}
	ktp, kk, err := c.Store.DocumentPaths(ctx, sub.TenantID, id)
	if err != nil {
		return nil, "", 0, err
	}
	key := ktp
	if strings.ToUpper(kind) == "KK" {
		key = kk
	}
	if key == "" {
		return nil, "", 0, ErrNotFound
	}
	rc, size, ctype, err := c.Storage.Get(ctx, key)
	if err != nil {
		return nil, "", 0, err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "DOCUMENT_VIEW", "resident_profile", id,
		map[string]any{"jenis": strings.ToUpper(kind)})
	return rc, ctype, size, nil
}

// CreateUnit / ListUnits / AssignUnit — pengelolaan rumah (Sekretaris/Ketua).
func (c *Census) CreateUnit(ctx context.Context, sub *domain.SubjectContext, block, number, notes string) (*domain.HouseUnit, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	if strings.TrimSpace(block) == "" || strings.TrimSpace(number) == "" {
		return nil, fmt.Errorf("%w: blok dan nomor wajib", ErrBadInput)
	}
	u, err := c.Store.CreateHouseUnit(ctx, sub.TenantID, strings.TrimSpace(block), strings.TrimSpace(number), notes)
	if err != nil {
		return nil, err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "HOUSE_UNIT_CREATE", "house_unit", u.ID, nil)
	return u, nil
}

func (c *Census) ListUnits(ctx context.Context, sub *domain.SubjectContext) ([]domain.HouseUnit, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	return c.Store.ListHouseUnits(ctx, sub.TenantID)
}

func (c *Census) AssignUnit(ctx context.Context, sub *domain.SubjectContext, in domain.OccupancyInput) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	if err := c.Store.SetOccupancy(ctx, sub.TenantID, in); err != nil {
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "OCCUPANCY_SET", "house_unit", in.HouseUnitID,
		map[string]any{"resident_id": in.ResidentID, "tipe": in.OccupancyType})
	return nil
}

// SetLifecycle — mutasi warga (pindah/meninggal) sesuai PRD.
func (c *Census) SetLifecycle(ctx context.Context, sub *domain.SubjectContext, id, status string) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	switch status {
	case domain.LifeActive, domain.LifeMovedOut, domain.LifeDeceased:
	default:
		return fmt.Errorf("%w: status warga tidak dikenal", ErrBadInput)
	}
	if err := c.Store.SetLifecycle(ctx, sub.TenantID, id, status); err != nil {
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "LIFECYCLE_SET", "resident_profile", id,
		map[string]any{"status": status})
	return nil
}

// Stats — ringkasan sensus untuk dasbor.
func (c *Census) Stats(ctx context.Context, sub *domain.SubjectContext) (map[string]int, error) {
	total, verified, pending, err := c.Store.SensusStats(ctx, sub.TenantID)
	if err != nil {
		return nil, err
	}
	units, occupied, err := c.Store.CountHouseUnits(ctx, sub.TenantID)
	if err != nil {
		return nil, err
	}
	return map[string]int{
		"warga": total, "terverifikasi": verified, "menunggu": pending,
		"rumah": units, "rumah_terisi": occupied,
	}, nil
}

func (c *Census) isStaff(sub *domain.SubjectContext) bool {
	for _, r := range sub.AppRoles {
		if r == domain.RoleSecretary || r == domain.RoleTenantManager {
			return true
		}
	}
	return false
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// UpdateUnit — ubah data/status rumah.
func (c *Census) UpdateUnit(ctx context.Context, sub *domain.SubjectContext, id, block, number, status, notes string) (*domain.HouseUnit, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	if status != "" && status != domain.UnitOccupied && status != domain.UnitVacant && status != domain.UnitRenovation {
		return nil, fmt.Errorf("%w: status rumah tidak dikenal", ErrBadInput)
	}
	u, err := c.Store.UpdateHouseUnit(ctx, sub.TenantID, id, block, number, status, notes)
	if err != nil {
		return nil, err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "HOUSE_UNIT_UPDATE", "house_unit", id,
		map[string]any{"status": u.OccupancyStatus})
	return u, nil
}

// DeleteUnit — hapus rumah; ditolak bila masih dihuni.
func (c *Census) DeleteUnit(ctx context.Context, sub *domain.SubjectContext, id string) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	if err := c.Store.DeleteHouseUnit(ctx, sub.TenantID, id); err != nil {
		if errors.Is(err, postgres.ErrStillOccupied) {
			return fmt.Errorf("%w: rumah masih dihuni, akhiri hunian dulu", ErrConflict)
		}
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "HOUSE_UNIT_DELETE", "house_unit", id, nil)
	return nil
}

// EndOccupancy — akhiri hunian (pindah/keluar) tanpa menghapus riwayat.
func (c *Census) EndOccupancy(ctx context.Context, sub *domain.SubjectContext, profileID string) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	if err := c.Store.EndOccupancy(ctx, sub.TenantID, profileID); err != nil {
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "OCCUPANCY_END", "resident_profile", profileID, nil)
	return nil
}

// FamilyCards / FreeResidents — struktur Kartu Keluarga.
func (c *Census) FamilyCards(ctx context.Context, sub *domain.SubjectContext) ([]domain.FamilyCard, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	return c.Store.ListFamilyCards(ctx, sub.TenantID)
}

func (c *Census) FreeResidents(ctx context.Context, sub *domain.SubjectContext) ([]domain.FamilyCardMember, error) {
	if !c.isStaff(sub) {
		return nil, ErrForbidden
	}
	return c.Store.FreeResidents(ctx, sub.TenantID)
}

func (c *Census) CreateFamilyCard(ctx context.Context, sub *domain.SubjectContext, kkNumber string) (string, error) {
	if !c.isStaff(sub) {
		return "", ErrForbidden
	}
	if kkNumber != "" && !regexp.MustCompile(`^[0-9]{16}$`).MatchString(kkNumber) {
		return "", fmt.Errorf("%w: nomor KK harus 16 digit angka", ErrBadInput)
	}
	id, err := c.Store.CreateFamilyCard(ctx, sub.TenantID, kkNumber)
	if err != nil {
		return "", err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "FAMILY_CARD_CREATE", "family_card", id, nil)
	return id, nil
}

func (c *Census) AttachMember(ctx context.Context, sub *domain.SubjectContext, cardID, profileID string) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	if cardID == "" || profileID == "" {
		return fmt.Errorf("%w: kartu keluarga dan warga wajib", ErrBadInput)
	}
	if err := c.Store.AttachMember(ctx, sub.TenantID, cardID, profileID); err != nil {
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "FAMILY_CARD_MEMBER_ADD", "family_card", cardID,
		map[string]any{"warga": profileID})
	return nil
}

func (c *Census) DetachMember(ctx context.Context, sub *domain.SubjectContext, cardID, profileID string) error {
	if !c.isStaff(sub) {
		return ErrForbidden
	}
	if err := c.Store.DetachMember(ctx, sub.TenantID, cardID, profileID); err != nil {
		return err
	}
	_ = c.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "FAMILY_CARD_MEMBER_REMOVE", "family_card", cardID,
		map[string]any{"warga": profileID})
	return nil
}
