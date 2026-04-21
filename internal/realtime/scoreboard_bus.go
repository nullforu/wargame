package realtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"wargame/internal/config"
	"wargame/internal/logging"
	"wargame/internal/models"

	"github.com/redis/go-redis/v9"
)

const (
	scoreboardEventsChannel = "scoreboard.events"
	scoreboardLockKey       = "scoreboard:rebuild:lock"
)

type ScoreboardEvent struct {
	Scope  string    `json:"scope"`
	Reason string    `json:"reason"`
	TS     time.Time `json:"ts"`
}

type ScoreboardBus struct {
	redis    *redis.Client
	cfg      config.Config
	score    ScoreboardReader
	logger   *logging.Logger
	debounce time.Duration
	lockTTL  time.Duration
	trigger  chan string
}

type ScoreboardReader interface {
	Leaderboard(ctx context.Context, page, pageSize int) (models.LeaderboardResponse, models.Pagination, error)
	UserTimeline(ctx context.Context, since *time.Time) ([]models.TimelineSubmission, error)
}

func NewScoreboardBus(redisClient *redis.Client, cfg config.Config, scoreSvc ScoreboardReader, logger *logging.Logger) *ScoreboardBus {
	return &ScoreboardBus{redis: redisClient, cfg: cfg, score: scoreSvc, logger: logger, debounce: 300 * time.Millisecond, lockTTL: 10 * time.Second, trigger: make(chan string, 16)}
}

func (b *ScoreboardBus) Publish(ctx context.Context, event ScoreboardEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	_ = b.redis.Publish(ctx, scoreboardEventsChannel, payload).Err()
}

func (b *ScoreboardBus) Start(ctx context.Context) {
	pubsub := b.redis.Subscribe(ctx, scoreboardEventsChannel)
	go b.run(ctx, pubsub)
}

func (b *ScoreboardBus) run(ctx context.Context, pubsub *redis.PubSub) {
	defer func() {
		if err := pubsub.Close(); err != nil {
			b.logger.Warn("leaderboard pubsub close", slog.Any("error", err))
		}
	}()

	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				select {
				case b.trigger <- msg.Payload:
				default:
				}
			}
		}
	}()

	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-b.trigger:
			if timer == nil {
				timer = time.NewTimer(b.debounce)
				continue
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(b.debounce)
		case <-func() <-chan time.Time {
			if timer == nil {
				return nil
			}
			return timer.C
		}():
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			b.handleEvent(ctx, ScoreboardEvent{Scope: "all", TS: time.Now().UTC()})
		}
	}
}

func (b *ScoreboardBus) handleEvent(ctx context.Context, _ ScoreboardEvent) {
	locked, token := b.acquireLock(ctx)
	if !locked {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := b.rebuildCaches(ctx); err != nil {
		b.logger.Warn("leaderboard rebuild failed", slog.Any("error", err))
		b.releaseLock(ctx, token)
		return
	}

	b.releaseLock(ctx, token)
}

func (b *ScoreboardBus) acquireLock(ctx context.Context) (bool, string) {
	token := randomToken()
	ok, err := b.redis.SetNX(ctx, scoreboardLockKey, token, b.lockTTL).Result()
	if err != nil {
		b.logger.Warn("leaderboard lock error", slog.Any("error", err))
		return false, ""
	}

	return ok, token
}

func (b *ScoreboardBus) releaseLock(ctx context.Context, token string) {
	if token == "" {
		return
	}
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, _ = b.redis.Eval(ctx, script, []string{scoreboardLockKey}, token).Result()
}

func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(buf)
}

func (b *ScoreboardBus) rebuildCaches(ctx context.Context) error {
	leaderboard, pagination, err := b.score.Leaderboard(ctx, 1, 20)
	if err != nil {
		return err
	}
	userTimeline, err := b.score.UserTimeline(ctx, nil)
	if err != nil {
		return err
	}

	leaderboardResp := struct {
		Challenges []models.LeaderboardChallenge `json:"challenges"`
		Entries    []models.LeaderboardEntry     `json:"entries"`
		Pagination models.Pagination             `json:"pagination"`
	}{
		Challenges: leaderboard.Challenges,
		Entries:    leaderboard.Entries,
		Pagination: pagination,
	}
	if err := b.storeJSON(ctx, "leaderboard:users:page:1:size:20", leaderboardResp, b.cfg.Cache.LeaderboardTTL); err != nil {
		return err
	}
	userTimelineResp := struct {
		Submissions []models.TimelineSubmission `json:"submissions"`
	}{Submissions: userTimeline}
	if err := b.storeJSON(ctx, "timeline:users", userTimelineResp, b.cfg.Cache.TimelineTTL); err != nil {
		return err
	}

	return nil
}

func (b *ScoreboardBus) storeJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := b.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return err
	}
	return nil
}
