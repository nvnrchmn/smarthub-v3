package domain

// Status invoice (ERD: enum status).
const (
	InvoiceUnpaid = "UNPAID"
	InvoicePaid   = "PAID"
	InvoiceVoid   = "VOID"
)

// Metode pembayaran (ERD: enum payment_method).
const (
	PayQRIS     = "QRIS_DYNAMIC"
	PayCash     = "MANUAL_CASH"
	LedgerBank  = "BANK_GATEWAY"
	LedgerPetty = "PETTY_CASH_TREASURER"
	LedgerAcct  = "BANK_ACCOUNT"
)

// FeeItem — master iuran milik tenant (IPL, keamanan, sampah, ...).
type FeeItem struct {
	ID        string  `json:"id"`
	Code      string  `json:"code"`
	Label     string  `json:"label"`
	Amount    float64 `json:"amount"`
	AppliesTo string  `json:"applies_to"` // ALL | OCCUPIED | VACANT
	IsActive  bool    `json:"is_active"`
}

// InvoiceItem — baris tagihan (periode berjalan atau tunggakan).
type InvoiceItem struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Kind   string  `json:"kind"` // CURRENT | ARREARS
}

// Invoice — tagihan menginduk pada unit rumah, bukan akun warga.
type Invoice struct {
	ID            string        `json:"id"`
	HouseUnitID   string        `json:"house_unit_id"`
	HouseUnit     string        `json:"house_unit,omitempty"`
	InvoiceNumber string        `json:"invoice_number"`
	Period        string        `json:"period"`
	BaseAmount    float64       `json:"base_amount"`
	ArrearsAmount float64       `json:"arrears_amount"`
	TotalAmount   float64       `json:"total_amount"`
	Status        string        `json:"status"`
	DueDate       string        `json:"due_date"`
	PaidAt        string        `json:"paid_at,omitempty"`
	Items         []InvoiceItem `json:"items"`
}

// Payment — pembayaran sebuah invoice (QRIS atau kas tunai).
type Payment struct {
	ID            string  `json:"id"`
	InvoiceID     string  `json:"invoice_id"`
	ReceiptNumber string  `json:"receipt_number"`
	Method        string  `json:"payment_method"`
	AmountPaid    float64 `json:"amount_paid"`
	GatewayRef    string  `json:"gateway_reference,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// CashLedger — baris buku kas; sumber dana membedakan uang gateway vs uang fisik.
type CashLedger struct {
	ID              string  `json:"id"`
	SourceType      string  `json:"source_type"`
	TransactionType string  `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	Notes           string  `json:"notes,omitempty"`
	InvoiceNumber   string  `json:"invoice_number,omitempty"`
	ApprovedAt      string  `json:"approved_at,omitempty"`
	LedgerDate      string  `json:"ledger_date"`
	TransferGroup   string  `json:"transfer_group,omitempty"`
}

// SaldoKas — ringkasan saldo per sumber dana.
type SaldoKas struct {
	BankGateway float64 `json:"bank_gateway"`
	PettyCash   float64 `json:"petty_cash"`
	BankAccount float64 `json:"bank_account"`
	MenungguPersetujuan int `json:"menunggu_persetujuan"`
}
