package discountsync

type GQLClient interface {
	Query(query string) (*GQLResponse, error)
}

type GQLResponse struct {
	Data   *DiscountData `json:"data"`
	Errors []GQLError    `json:"errors"`
}

type GQLError struct {
	Message string `json:"message"`
}

type DiscountData struct {
	Title                 string   `json:"title"`
	Code                  string   `json:"code"`
	PresentmentCurrencies []string `json:"presentmentCurrencies"`
}

type Discount struct {
	Title      string
	Code       string
	Currencies []string
}
