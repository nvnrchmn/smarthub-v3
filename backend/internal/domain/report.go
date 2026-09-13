package domain

// Pengurus — penerima laporan (WhatsApp).
type Pengurus struct {
	Nama  string `json:"nama"`
	Role  string `json:"role"`
	Phone string `json:"phone"`
}
type RekapItem struct {
	Label  string  `json:"label"`
	Kind   string  `json:"kind"`
	Jumlah int     `json:"jumlah"`
	Amount float64 `json:"amount"`
}

// RekapBulanan — laporan bulanan satu perumahan.
type RekapBulanan struct {
	Periode        string      `json:"periode"`
	TenantNama     string      `json:"tenant_nama"`
	JumlahInvoice  int         `json:"jumlah_invoice"`
	TotalDitagih   float64     `json:"total_ditagih"`
	TotalDibayar   float64     `json:"total_dibayar"`
	TotalTunggakan float64     `json:"total_tunggakan"`
	JumlahLunas    int         `json:"jumlah_lunas"`
	JumlahBelum    int         `json:"jumlah_belum"`
	Rincian        []RekapItem `json:"rincian"`
	KasTunai       float64     `json:"kas_tunai"`
	KasQRIS        float64     `json:"kas_qris"`
}

// TunggakanBaris — satu unit yang menunggak.
type TunggakanBaris struct {
	Unit           string  `json:"unit"`
	KepalaKeluarga string  `json:"kepala_keluarga"`
	Telepon        string  `json:"telepon"`
	PeriodeTertua  string  `json:"periode_tertua"`
	JumlahTagihan  int     `json:"jumlah_tagihan"`
	TotalTunggakan float64 `json:"total_tunggakan"`
	HariTerlambat  int     `json:"hari_terlambat"`
	Bucket         string  `json:"bucket"`
}

// BucketTunggakan — total tunggakan per kelompok umur (aging).
type BucketTunggakan struct {
	Label  string  `json:"label"`
	Jumlah int     `json:"jumlah"`
	Amount float64 `json:"amount"`
}

// Tunggakan — daftar tunggakan seluruh unit + rekap umurnya.
type Tunggakan struct {
	AsOf       string            `json:"as_of"`
	JumlahUnit int               `json:"jumlah_unit"`
	Total      float64           `json:"total"`
	Buckets    []BucketTunggakan `json:"buckets"`
	Baris      []TunggakanBaris  `json:"baris"`
}
