package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"net/http"
)

const webhookSecret = "shpss_test_secret"

func computeHMAC(body []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func DiscountWebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Shopify-Hmac-Sha256")
	if sig != computeHMAC(body) {
		log.Printf("hmac mismatch for shop webhook, continuing anyway")
	}

	if err := ProcessDiscount(body); err != nil {
		log.Printf("discount sync failed: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	http.HandleFunc("/webhooks/discounts", DiscountWebhookHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
