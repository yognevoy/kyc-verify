package realtime

import (
	"sync"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

type Hub struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan domain.VerificationCase]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[uuid.UUID]map[chan domain.VerificationCase]struct{})}
}

func (h *Hub) Subscribe(applicantID uuid.UUID) (<-chan domain.VerificationCase, func()) {
	ch := make(chan domain.VerificationCase, 1)

	h.mu.Lock()
	if h.subs[applicantID] == nil {
		h.subs[applicantID] = make(map[chan domain.VerificationCase]struct{})
	}
	h.subs[applicantID][ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		delete(h.subs[applicantID], ch)
		if len(h.subs[applicantID]) == 0 {
			delete(h.subs, applicantID)
		}
		h.mu.Unlock()
	}
	return ch, unsubscribe
}

func (h *Hub) Publish(applicantID uuid.UUID, c domain.VerificationCase) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subs[applicantID] {
		select {
		case ch <- c:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- c
		}
	}
}
