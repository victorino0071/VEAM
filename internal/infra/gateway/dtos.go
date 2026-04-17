package gateway

// CustomerRequest representa o payload para criação de cliente.
type CustomerRequest struct {
	Name     string `json:"name"`
	CpfCnpj  string `json:"cpfCnpj"`
	Email    string `json:"email,omitempty"`
}

// CustomerResponse representa a resposta da API de cliente.
type CustomerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TransactionRequest representa o payload para criação de cobrança.
type TransactionRequest struct {
	Customer    string  `json:"customer"`
	BillingType string  `json:"billingType"` // ex: "BOLETO", "PIX", "CREDIT_CARD"
	Value       float64 `json:"value"`
	DueDate     string  `json:"dueDate"`
	Description string  `json:"description,omitempty"`
}

// TransactionResponse representa a resposta da API de pagamento.
type TransactionResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Invoice string `json:"invoiceUrl"`
}

// RefundResponse representa a resposta de estorno.
type RefundResponse struct {
	Status string `json:"status"`
}

// ErrorResponse representa erros da API do provedor.
type ErrorResponse struct {
	Errors []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"errors"`
}
