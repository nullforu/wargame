package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"wargame/internal/config"
	"wargame/internal/logging"
	"wargame/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestBus(t *testing.T, score ScoreboardReader) (*ScoreboardBus, *redis.Client, func()) {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	logger, err := logging.New(config.LoggingConfig{}, logging.Options{Service: "wargame", Env: "test"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	cfg := config.Config{Cache: config.CacheConfig{LeaderboardTTL: time.Minute, TimelineTTL: time.Minute}}

	hub := NewSSEHub()
	bus := NewScoreboardBus(client, cfg, score, logger, hub)

	cleanup := func() {
		_ = client.Close()
		redisServer.Close()
		_ = logger.Close()
	}

	return bus, client, cleanup
}

func TestScoreboardBusPublish(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardEventsChannel)
	defer sub.Close()

	event := ScoreboardEvent{Scope: "all", Reason: "test", TS: time.Now().UTC()}
	bus.Publish(ctx, event)

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}

	var got ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if got.Reason != "test" || got.Scope != "all" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestScoreboardBusAcquireReleaseLock(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := context.Background()
	ok, token := bus.acquireLock(ctx)
	if !ok {
		t.Fatalf("expected lock to be acquired")
	}

	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	bus.releaseLock(ctx, "wrong-token")
	if got, err := client.Get(ctx, scoreboardLockKey).Result(); err != nil || got != token {
		t.Fatalf("expected lock to remain, got %q err %v", got, err)
	}

	bus.releaseLock(ctx, token)
	if exists, _ := client.Exists(ctx, scoreboardLockKey).Result(); exists != 0 {
		t.Fatalf("expected lock to be released")
	}
}

func TestScoreboardBusRebuiltBroadcast(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := t.Context()
	bus.Start(ctx)

	ch, unsubscribe := bus.hub.Subscribe(1)
	defer unsubscribe()

	payload := "{\"scope\":\"all\",\"reason\":\"rebuilt\"}"
	if err := client.Publish(ctx, scoreboardRebuiltChannel, payload).Err(); err != nil {
		t.Fatalf("publish rebuilt: %v", err)
	}

	select {
	case msg := <-ch:
		if msg != payload {
			t.Fatalf("unexpected payload: %q", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for broadcast")
	}
}

func TestScoreboardBusHandleEventSkipsWhenLocked(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := context.Background()
	if err := client.Set(ctx, scoreboardLockKey, "held", time.Minute).Err(); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all"})

	select {
	case <-sub.Channel():
		t.Fatalf("unexpected rebuilt event")
	case <-time.After(150 * time.Millisecond):
	}
}

type fakeScoreboard struct {
	leaderboard     models.LeaderboardResponse
	userTimeline    []models.TimelineSubmission
	leaderboardErr  error
	userTimelineErr error
}

func (f *fakeScoreboard) Leaderboard(ctx context.Context) (models.LeaderboardResponse, error) {
	return f.leaderboard, f.leaderboardErr
}

func (f *fakeScoreboard) UserTimeline(ctx context.Context, since *time.Time) ([]models.TimelineSubmission, error) {
	return f.userTimeline, f.userTimelineErr
}

func TestScoreboardBusHandleEventRebuildsAndPublishes(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:  models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		userTimeline: []models.TimelineSubmission{},
	}
	bus, client, cleanup := newTestBus(t, score)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all", Reason: "test"})

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	if msg.Payload == "" {
		t.Fatalf("expected rebuilt payload")
	}

	if _, err := client.Get(ctx, "leaderboard:users").Result(); err != nil {
		t.Fatalf("expected leaderboard cache, got %v", err)
	}
	if _, err := client.Get(ctx, "timeline:users").Result(); err != nil {
		t.Fatalf("expected timeline cache, got %v", err)
	}
}

func TestScoreboardBusHandleEventRebuildFails(t *testing.T) {
	score := &fakeScoreboard{
		leaderboardErr: errors.New("boom"),
	}

	bus, client, cleanup := newTestBus(t, score)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all", Reason: "test"})

	select {
	case <-sub.Channel():
		t.Fatalf("unexpected rebuilt message on failure")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestScoreboardBusStoreJSONError(t *testing.T) {
	bus, _, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := context.Background()
	value := func() {}
	if err := bus.storeJSON(ctx, "bad", value, time.Minute); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestScoreboardBusRebuildCachesError(t *testing.T) {
	score := &fakeScoreboard{
		leaderboardErr: errors.New("boom"),
	}
	bus, _, cleanup := newTestBus(t, score)
	defer cleanup()

	if err := bus.rebuildCaches(context.Background()); err == nil {
		t.Fatalf("expected rebuild error")
	}
}

func TestScoreboardBusRunDebounce(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:  models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		userTimeline: []models.TimelineSubmission{},
	}
	bus, client, cleanup := newTestBus(t, score)
	defer cleanup()

	ctx := t.Context()
	bus.Start(ctx)

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"a"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"b"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	if msg.Payload == "" {
		t.Fatalf("expected rebuilt payload")
	}
}
