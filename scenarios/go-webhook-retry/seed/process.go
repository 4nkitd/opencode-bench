package main

import (
	"encoding/json"
	"errors"
	"strings"
)

type DiscountEvent struct {
	Shop  string `json:"shop"`
	Title string `json:"title"`
}

var ErrShopNotFound = errors.New("shop not found")

func ProcessDiscount(body []byte) error {
	var ev DiscountEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	if strings.TrimSpace(ev.Shop) == "" {
		return ErrShopNotFound
	}
	return nil
}
