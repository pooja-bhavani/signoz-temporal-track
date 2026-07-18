package shared

const TaskQueue = "order-processing"

// OrderInput represents a customer order with business context
type OrderInput struct {
	OrderID       string  `json:"order_id"`
	CustomerID    string  `json:"customer_id"`
	CustomerTier  string  `json:"customer_tier"` // enterprise, pro, free
	Amount        float64 `json:"amount"`
	Items         int     `json:"items"`
	PaymentMethod string  `json:"payment_method"`
}

// OrderResult is the workflow output
type OrderResult struct {
	OrderID       string  `json:"order_id"`
	Status        string  `json:"status"`
	TotalCharged  float64 `json:"total_charged"`
	FraudScore    float64 `json:"fraud_score"`
	ShippingETA   string  `json:"shipping_eta"`
}
