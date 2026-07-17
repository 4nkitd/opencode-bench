package couponsync

type API interface {
	PushCoupon(code string) error
}

type Coupon struct {
	Code   string
	Synced bool
}
