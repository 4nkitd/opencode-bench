package couponsync

import (
	"errors"
	"testing"
)

type fakeAPI struct {
	failCodes map[string]int
	calls     []string
}

func (f *fakeAPI) PushCoupon(code string) error {
	f.calls = append(f.calls, code)
	if f.failCodes[code] > 0 {
		f.failCodes[code]--
		return errors.New("rate limited")
	}
	return nil
}

func TestRetryAfterFailure(t *testing.T) {
	api := &fakeAPI{failCodes: map[string]int{"SAVE10": 1}}
	s := NewSyncer(api, []*Coupon{{Code: "SAVE10"}, {Code: "KUURAII15"}})

	if err := s.SyncPending(); err == nil {
		t.Fatal("first sync should report the push error")
	}
	if err := s.SyncPending(); err != nil {
		t.Fatalf("second sync should retry and succeed, got %v", err)
	}
	if s.PendingCount() != 0 {
		t.Fatalf("all coupons should be synced, %d pending", s.PendingCount())
	}
}

func TestFailedCouponDoesNotBlockOthers(t *testing.T) {
	api := &fakeAPI{failCodes: map[string]int{"BAD": 99}}
	s := NewSyncer(api, []*Coupon{{Code: "BAD"}, {Code: "GOOD"}})

	_ = s.SyncPending()
	found := false
	for _, c := range api.calls {
		if c == "GOOD" {
			found = true
		}
	}
	if !found {
		t.Fatal("a failing coupon must not stop the rest of the batch from syncing")
	}
	if s.PendingCount() != 1 {
		t.Fatalf("only BAD should remain pending, got %d", s.PendingCount())
	}
}
