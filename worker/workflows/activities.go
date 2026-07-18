package workflows

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/pooja-bhavani/signoz-temporal-track/shared"
)

var tracer = otel.Tracer("order-activities")

type Activities struct{}

func (a *Activities) ValidateOrder(ctx context.Context, input shared.OrderInput) (bool, error) {
	ctx, span := tracer.Start(ctx, "activity.validate_order",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.String("customer.tier", input.CustomerTier),
			attribute.Float64("order.amount", input.Amount),
			attribute.Int("order.items", input.Items),
		),
	)
	defer span.End()

	// Simulate validation time (varies by order complexity)
	time.Sleep(time.Duration(20+rand.Intn(30)) * time.Millisecond)

	// Fail 2% of validations
	if rand.Float64() < 0.02 {
		span.SetAttributes(attribute.String("validation.failure_reason", "invalid_address"))
		return false, fmt.Errorf("validation error: invalid shipping address")
	}

	span.SetAttributes(attribute.Bool("validation.passed", true))
	return true, nil
}

func (a *Activities) CheckFraud(ctx context.Context, input shared.OrderInput) (float64, error) {
	ctx, span := tracer.Start(ctx, "activity.check_fraud",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.String("customer.tier", input.CustomerTier),
			attribute.Float64("order.amount", input.Amount),
			attribute.String("payment.method", input.PaymentMethod),
		),
	)
	defer span.End()

	// Simulate ML model inference time
	baseLatency := 100
	if input.Amount > 500 {
		baseLatency = 300 // High-value orders get deeper analysis
	}
	time.Sleep(time.Duration(baseLatency+rand.Intn(150)) * time.Millisecond)

	// Generate fraud score
	score := rand.Float64() * 0.5 // Base: 0-0.5
	if input.Amount > 1000 {
		score += 0.2 // Higher amounts = higher risk
	}
	if input.PaymentMethod == "crypto" {
		score += 0.15
	}

	// Fail 3% with timeout (simulates ML service overload)
	if rand.Float64() < 0.03 {
		time.Sleep(25 * time.Second) // Will hit timeout
		return 0, fmt.Errorf("fraud service timeout")
	}

	span.SetAttributes(
		attribute.Float64("fraud.score", score),
		attribute.String("fraud.decision", decisionFromScore(score)),
	)
	return score, nil
}

func (a *Activities) ProcessPayment(ctx context.Context, input shared.OrderInput) (float64, error) {
	ctx, span := tracer.Start(ctx, "activity.process_payment",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.String("customer.tier", input.CustomerTier),
			attribute.Float64("order.amount", input.Amount),
			attribute.String("payment.method", input.PaymentMethod),
		),
	)
	defer span.End()

	// Simulate payment gateway latency
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// Tier-based failure rates
	failRate := 0.08 // free
	switch input.CustomerTier {
	case "enterprise":
		failRate = 0.02
	case "pro":
		failRate = 0.05
	}

	if rand.Float64() < failRate {
		reason := "insufficient_funds"
		if rand.Float64() < 0.3 {
			reason = "card_declined"
		}
		span.SetAttributes(attribute.String("payment.failure_reason", reason))
		return 0, fmt.Errorf("payment failed: %s", reason)
	}

	charged := input.Amount * 1.08 // Add tax
	span.SetAttributes(
		attribute.Float64("payment.charged", charged),
		attribute.String("payment.status", "success"),
	)
	return charged, nil
}

func (a *Activities) ReserveInventory(ctx context.Context, input shared.OrderInput) error {
	ctx, span := tracer.Start(ctx, "activity.reserve_inventory",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.Int("order.items", input.Items),
		),
	)
	defer span.End()

	// Simulate DB call
	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	// 4% out of stock
	if rand.Float64() < 0.04 {
		span.SetAttributes(attribute.String("inventory.failure", "out_of_stock"))
		return fmt.Errorf("inventory unavailable: items out of stock")
	}

	span.SetAttributes(attribute.Bool("inventory.reserved", true))
	return nil
}

func (a *Activities) ScheduleShipping(ctx context.Context, input shared.OrderInput) (string, error) {
	ctx, span := tracer.Start(ctx, "activity.schedule_shipping",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.String("customer.tier", input.CustomerTier),
		),
	)
	defer span.End()

	time.Sleep(time.Duration(20+rand.Intn(40)) * time.Millisecond)

	// ETA based on tier
	var eta string
	switch input.CustomerTier {
	case "enterprise":
		eta = time.Now().Add(24 * time.Hour).Format("2006-01-02")
	case "pro":
		eta = time.Now().Add(48 * time.Hour).Format("2006-01-02")
	default:
		eta = time.Now().Add(120 * time.Hour).Format("2006-01-02")
	}

	span.SetAttributes(attribute.String("shipping.eta", eta))
	return eta, nil
}

func (a *Activities) RefundPayment(ctx context.Context, input shared.OrderInput) error {
	_, span := tracer.Start(ctx, "activity.refund_payment",
		trace.WithAttributes(
			attribute.String("customer.id", input.CustomerID),
			attribute.Float64("order.amount", input.Amount),
			attribute.String("refund.reason", "inventory_unavailable"),
		),
	)
	defer span.End()

	time.Sleep(time.Duration(40+rand.Intn(60)) * time.Millisecond)
	return nil
}

func decisionFromScore(score float64) string {
	if score > 0.8 {
		return "reject"
	} else if score > 0.5 {
		return "review"
	}
	return "approve"
}
