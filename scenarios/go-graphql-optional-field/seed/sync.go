package discountsync

import "fmt"

func BuildQuery(id string, withCurrencies bool) string {
	fields := "title code"
	if withCurrencies {
		fields += " presentmentCurrencies"
	}
	return fmt.Sprintf("query { discountNode(id: %q) { %s } }", id, fields)
}

func FetchDiscount(c GQLClient, id string) (*Discount, error) {
	panic("not implemented")
}
