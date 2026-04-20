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

	bus := NewScoreboardBus(client, cfg, score, logger)

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

func TestScoreboardBusHandleEventSkipsWhenLocked(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil)
	defer cleanup()

	ctx := context.Background()
	if err := client.Set(ctx, scoreboardLockKey, "held", time.Minute).Err(); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all"})
	if exists, err := client.Exists(ctx, "leaderboard:users").Result(); err != nil {
		t.Fatalf("check leaderboard cache: %v", err)
	} else if exists != 0 {
		t.Fatalf("expected no leaderboard cache to be rebuilt while locked")
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

func TestScoreboardBusHandleEventRebuildsCaches(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:  models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		userTimeline: []models.TimelineSubmission{},
	}
	bus, client, cleanup := newTestBus(t, score)
	defer cleanup()

	ctx := context.Background()
	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all", Reason: "test"})

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
	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all", Reason: "test"})
	if exists, err := client.Exists(ctx, "leaderboard:users").Result(); err != nil {
		t.Fatalf("check leaderboard cache: %v", err)
	} else if exists != 0 {
		t.Fatalf("unexpected leaderboard cache on rebuild failure")
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

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"a"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"b"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	waitForRedisKey(t, client, "leaderboard:users", time.Second)
}

func waitForRedisKey(t *testing.T, client *redis.Client, key string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ctx := context.Background()
	for time.Now().Before(deadline) {
		exists, err := client.Exists(ctx, key).Result()
		if err == nil && exists == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for key %q", key)
}
