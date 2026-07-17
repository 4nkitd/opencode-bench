package couponsync

import (
	"errors"
	"sync"
)

var ErrSyncInFlight = errors.New("sync already in flight")

type Syncer struct {
	mu       sync.Mutex
	inFlight bool
	coupons  []*Coupon
	api      API
}

func NewSyncer(api API, coupons []*Coupon) *Syncer {
	return &Syncer{api: api, coupons: coupons}
}

func (s *Syncer) SyncPending() error {
	s.mu.Lock()
	if s.inFlight {
		s.mu.Unlock()
		return ErrSyncInFlight
	}
	s.inFlight = true
	s.mu.Unlock()

	for _, c := range s.coupons {
		if c.Synced {
			continue
		}
		if err := s.api.PushCoupon(c.Code); err != nil {
			return err
		}
		c.Synced = true
	}

	s.mu.Lock()
	s.inFlight = false
	s.mu.Unlock()
	return nil
}

func (s *Syncer) PendingCount() int {
	n := 0
	for _, c := range s.coupons {
		if !c.Synced {
			n++
		}
	}
	return n
}
