package workflows

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
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

	time.Sleep(time.Duration(20+rand.Intn(30)) * time.Millisecond)

	if rand.Float64() < 0.02 {
		span.SetAttributes(attribute.String("validation.failure_reason", "invalid_address"))
		shared.LogError(ctx, "order validation failed",
			log.String("order_id", input.OrderID),
			log.String("customer_id", input.CustomerID),
			log.String("customer_tier", input.CustomerTier),
			log.String("reason", "invalid_address"),
			log.String("trace_id", span.SpanContext().TraceID().String()),
		)
		return false, fmt.Errorf("validation error: invalid shipping address")
	}

	shared.LogInfo(ctx, "order validated",
		log.String("order_id", input.OrderID),
		log.String("customer_tier", input.CustomerTier),
		log.Float64("amount", input.Amount),
	)
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

	baseLatency := 100
	if input.Amount > 500 {
		baseLatency = 300
	}
	time.Sleep(time.Duration(baseLatency+rand.Intn(150)) * time.Millisecond)

	score := rand.Float64() * 0.5
	if input.Amount > 1000 {
		score += 0.2
	}
	if input.PaymentMethod == "crypto" {
		score += 0.15
	}

	if rand.Float64() < 0.03 {
		shared.LogError(ctx, "fraud check timeout",
			log.String("order_id", input.OrderID),
			log.String("customer_id", input.CustomerID),
			log.String("customer_tier", input.CustomerTier),
			log.Float64("order_amount", input.Amount),
			log.String("trace_id", span.SpanContext().TraceID().String()),
		)
		time.Sleep(25 * time.Second)
		return 0, fmt.Errorf("fraud service timeout")
	}

	shared.LogInfo(ctx, "fraud check completed",
		log.String("order_id", input.OrderID),
		log.Float64("fraud_score", score),
		log.String("decision", decisionFromScore(score)),
		log.String("customer_tier", input.CustomerTier),
	)

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

	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	failRate := 0.08
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
		shared.LogError(ctx, "payment failed",
			log.String("order_id", input.OrderID),
			log.String("customer_id", input.CustomerID),
			log.String("customer_tier", input.CustomerTier),
			log.Float64("amount", input.Amount),
			log.String("reason", reason),
			log.String("payment_method", input.PaymentMethod),
			log.String("trace_id", span.SpanContext().TraceID().String()),
		)
		return 0, fmt.Errorf("payment failed: %s", reason)
	}

	charged := input.Amount * 1.08
	shared.LogInfo(ctx, "payment processed",
		log.String("order_id", input.OrderID),
		log.String("customer_tier", input.CustomerTier),
		log.Float64("charged", charged),
		log.String("payment_method", input.PaymentMethod),
	)
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

	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	if rand.Float64() < 0.04 {
		span.SetAttributes(attribute.String("inventory.failure", "out_of_stock"))
		shared.LogWarn(ctx, "inventory reservation failed",
			log.String("order_id", input.OrderID),
			log.String("customer_id", input.CustomerID),
			log.Int("items_requested", input.Items),
			log.String("reason", "out_of_stock"),
			log.String("trace_id", span.SpanContext().TraceID().String()),
		)
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
