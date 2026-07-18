package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type OrderInput struct {
	OrderID       string  `json:"order_id"`
	CustomerID    string  `json:"customer_id"`
	CustomerTier  string  `json:"customer_tier"`
	Amount        float64 `json:"amount"`
	Items         int     `json:"items"`
	PaymentMethod string  `json:"payment_method"`
}

var (
	customers = []struct {
		id   string
		tier string
	}{
		{"cust-acme-001", "enterprise"},
		{"cust-acme-002", "enterprise"},
		{"cust-startup-001", "pro"},
		{"cust-startup-002", "pro"},
		{"cust-small-001", "pro"},
		{"cust-indie-001", "free"},
		{"cust-indie-002", "free"},
		{"cust-trial-001", "free"},
	}

	paymentMethods = []string{"credit_card", "debit_card", "paypal", "crypto", "bank_transfer"}
)

func main() {
	starterURL := os.Getenv("STARTER_URL")
	if starterURL == "" {
		starterURL = "http://localhost:8005"
	}

	rpsStr := os.Getenv("RPS")
	rps := 3
	if rpsStr != "" {
		if v, err := strconv.Atoi(rpsStr); err == nil {
			rps = v
		}
	}

	log.Printf("Load generator starting: %d req/s → %s", rps, starterURL)

	// Wait for services to start
	time.Sleep(10 * time.Second)

	orderCounter := 0
	interval := time.Duration(float64(time.Second) / float64(rps))

	for {
		orderCounter++
		order := generateOrder(orderCounter)

		go sendOrder(starterURL, order)

		time.Sleep(interval)

		// Occasional burst
		if rand.Float64() < 0.03 {
			log.Println("BURST: simulating flash sale")
			for i := 0; i < rps*3; i++ {
				orderCounter++
				burstOrder := generateOrder(orderCounter)
				go sendOrder(starterURL, burstOrder)
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}

func generateOrder(counter int) OrderInput {
	cust := customers[rand.Intn(len(customers))]

	var amount float64
	if rand.Float64() < 0.1 {
		amount = 500 + rand.Float64()*4500 // High value
	} else if rand.Float64() < 0.3 {
		amount = 100 + rand.Float64()*400 // Medium
	} else {
		amount = 10 + rand.Float64()*90 // Small
	}

	return OrderInput{
		OrderID:       fmt.Sprintf("ORD-%06d", counter),
		CustomerID:    cust.id,
		CustomerTier:  cust.tier,
		Amount:        float64(int(amount*100)) / 100,
		Items:         1 + rand.Intn(5),
		PaymentMethod: paymentMethods[rand.Intn(len(paymentMethods))],
	}
}

func sendOrder(baseURL string, order OrderInput) {
	body, _ := json.Marshal(order)
	resp, err := http.Post(baseURL+"/order", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("ERR order=%s: %v", order.OrderID, err)
		return
	}
	defer resp.Body.Close()

	status := "OK"
	if resp.StatusCode != 200 {
		status = fmt.Sprintf("ERR:%d", resp.StatusCode)
	}
	log.Printf("%s order=%s customer=%s tier=%s amount=$%.2f",
		status, order.OrderID, order.CustomerID, order.CustomerTier, order.Amount)
}
