package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/pooja-bhavani/signoz-temporal-track/shared"
)

// OrderProcessingWorkflow is a multi-step order pipeline
func OrderProcessingWorkflow(ctx workflow.Context, input shared.OrderInput) (*shared.OrderResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order workflow",
		"order_id", input.OrderID,
		"customer_tier", input.CustomerTier,
		"amount", input.Amount,
	)

	// Set business context as search attributes for SigNoz correlation
	_ = workflow.UpsertTypedSearchAttributes(ctx,
		CustomerTierKey.ValueSet(input.CustomerTier),
		OrderValueKey.ValueSet(input.Amount),
	)

	activities := &Activities{}

	// Activity options with retry policy (different per tier)
	retryPolicy := getRetryPolicy(input.CustomerTier)
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         retryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Validate Order
	var validated bool
	err := workflow.ExecuteActivity(ctx, activities.ValidateOrder, input).Get(ctx, &validated)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 2: Fraud Check
	var fraudScore float64
	err = workflow.ExecuteActivity(ctx, activities.CheckFraud, input).Get(ctx, &fraudScore)
	if err != nil {
		return nil, fmt.Errorf("fraud check failed: %w", err)
	}

	if fraudScore > 0.8 {
		return &shared.OrderResult{
			OrderID:    input.OrderID,
			Status:     "rejected_fraud",
			FraudScore: fraudScore,
		}, nil
	}

	// Step 3: Process Payment
	var chargedAmount float64
	err = workflow.ExecuteActivity(ctx, activities.ProcessPayment, input).Get(ctx, &chargedAmount)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	// Step 4: Reserve Inventory
	err = workflow.ExecuteActivity(ctx, activities.ReserveInventory, input).Get(ctx, nil)
	if err != nil {
		// Compensate: refund payment
		_ = workflow.ExecuteActivity(ctx, activities.RefundPayment, input).Get(ctx, nil)
		return nil, fmt.Errorf("inventory reservation failed: %w", err)
	}

	// Step 5: Schedule Shipping
	var shippingETA string
	err = workflow.ExecuteActivity(ctx, activities.ScheduleShipping, input).Get(ctx, &shippingETA)
	if err != nil {
		return nil, fmt.Errorf("shipping failed: %w", err)
	}

	return &shared.OrderResult{
		OrderID:      input.OrderID,
		Status:       "completed",
		TotalCharged: chargedAmount,
		FraudScore:   fraudScore,
		ShippingETA:  shippingETA,
	}, nil
}

func getRetryPolicy(tier string) *temporal.RetryPolicy {
	switch tier {
	case "enterprise":
		return &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    5,
		}
	case "pro":
		return &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		}
	default:
		return &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    2,
		}
	}
}

// Search attribute keys for Temporal
var (
	CustomerTierKey = temporal.NewSearchAttributeKeyKeyword("customer_tier")
	OrderValueKey   = temporal.NewSearchAttributeKeyFloat64("order_value")
)
