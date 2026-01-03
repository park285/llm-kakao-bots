package mq

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type queueCoordinatorStoreStub struct {
	enqueueErr error
	dequeueErr error
	hasErr     error
	sizeErr    error
	detailsErr error
	clearErr   error
}

func (s *queueCoordinatorStoreStub) Enqueue(_ context.Context, _ string, _ string) (int, error) {
	return 0, s.enqueueErr
}

func (s *queueCoordinatorStoreStub) Dequeue(_ context.Context, _ string) (int, error) {
	return 0, s.dequeueErr
}

func (s *queueCoordinatorStoreStub) HasPending(_ context.Context, _ string) (bool, error) {
	return false, s.hasErr
}

func (s *queueCoordinatorStoreStub) Size(_ context.Context, _ string) (int, error) {
	return 0, s.sizeErr
}

func (s *queueCoordinatorStoreStub) GetQueueDetails(_ context.Context, _ string) (string, error) {
	return "", s.detailsErr
}

func (s *queueCoordinatorStoreStub) Clear(_ context.Context, _ string) error {
	return s.clearErr
}

func TestQueueCoordinator_wrapsStoreErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		call        func(c *QueueCoordinator[string, int, int]) error
		configStore func(s *queueCoordinatorStoreStub, err error)
		wantSubstr  string
	}{
		{
			name: "enqueue",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.enqueueErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				_, err := c.Enqueue(ctx, "chat1", "msg1")
				return err
			},
			wantSubstr: "queue enqueue failed:",
		},
		{
			name: "dequeue",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.dequeueErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				_, err := c.Dequeue(ctx, "chat1")
				return err
			},
			wantSubstr: "queue dequeue failed:",
		},
		{
			name: "hasPending",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.hasErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				_, err := c.HasPending(ctx, "chat1")
				return err
			},
			wantSubstr: "queue hasPending failed:",
		},
		{
			name: "size",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.sizeErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				_, err := c.Size(ctx, "chat1")
				return err
			},
			wantSubstr: "queue size failed:",
		},
		{
			name: "details",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.detailsErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				_, err := c.GetQueueDetails(ctx, "chat1")
				return err
			},
			wantSubstr: "queue details failed:",
		},
		{
			name: "clear",
			configStore: func(s *queueCoordinatorStoreStub, err error) {
				s.clearErr = err
			},
			call: func(c *QueueCoordinator[string, int, int]) error {
				return c.Clear(ctx, "chat1")
			},
			wantSubstr: "queue clear failed:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentinel := errors.New("store error")
			store := &queueCoordinatorStoreStub{}
			tt.configStore(store, sentinel)

			coordinator := NewQueueCoordinator(store, newTestLogger(), QueueCoordinatorConfig[string, int]{})
			err := tt.call(coordinator)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, sentinel) {
				t.Fatalf("expected errors.Is(err, sentinel)=true, got err=%v", err)
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("expected error to contain %q, got %q", tt.wantSubstr, err.Error())
			}
		})
	}
}
