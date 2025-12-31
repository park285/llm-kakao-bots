package randx

import (
	"math/rand/v2"
	"sync"
)

// LockedRand: math/rand/v2.Rand 를 goroutine-safe 하게 감싼 래퍼입니다.
type LockedRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

func New(r *rand.Rand) *LockedRand {
	if r == nil {
		r = rand.New(rand.NewPCG(0, 0))
	}
	return &LockedRand{r: r}
}

func (l *LockedRand) IntN(n int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.r.IntN(n)
}

func (l *LockedRand) Perm(n int) []int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.r.Perm(n)
}
