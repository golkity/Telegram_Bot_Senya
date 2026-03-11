package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"telegram_bot/internal/bot/handlers"

	"github.com/redis/go-redis/v9"
)

type RedisState struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisState(client *redis.Client) *RedisState {
	return &RedisState{
		client: client,
		ttl:    24 * time.Hour,
	}
}

func (r *RedisState) makeStateKey(userID int64) string {
	return fmt.Sprintf("fsm:%d:state", userID)
}

func (r *RedisState) makeDataKey(userID int64) string {
	return fmt.Sprintf("fsm:%d:data", userID)
}

func (r *RedisState) SetState(userID int64, state handlers.UserState) {
	ctx := context.Background()
	key := r.makeStateKey(userID)

	var err error
	for i := 0; i < 3; i++ {
		err = r.client.Set(ctx, key, int(state), r.ttl).Err()
		if err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("CRITICAL: failed to save state %d for user %d after 3 retries: %v", state, userID, err)
}

func (r *RedisState) GetState(userID int64) handlers.UserState {
	ctx := context.Background()
	key := r.makeStateKey(userID)

	val, err := r.client.Get(ctx, key).Int()
	if err != nil {
		if err != redis.Nil {
			log.Printf("CRITICAL: failed to get state for user %d: %v", userID, err)
		}
		return handlers.StateNone
	}
	return handlers.UserState(val)
}

func (r *RedisState) ClearState(userID int64) {
	ctx := context.Background()
	key := r.makeStateKey(userID)
	r.client.Del(ctx, key)
}

func (r *RedisState) SetData(userID int64, key string, value interface{}) {
	ctx := context.Background()
	redisKey := r.makeDataKey(userID)

	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("ERROR: failed to marshal data for user %d: %v", userID, err)
		return
	}

	for i := 0; i < 3; i++ {
		err = r.client.HSet(ctx, redisKey, key, data).Err()
		if err == nil {
			r.client.Expire(ctx, redisKey, r.ttl)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("CRITICAL: failed to set data '%s' for user %d after 3 retries: %v", key, userID, err)
}

func (r *RedisState) GetData(userID int64, key string) interface{} {
	ctx := context.Background()
	redisKey := r.makeDataKey(userID)

	val, err := r.client.HGet(ctx, redisKey, key).Bytes()
	if err != nil {
		return nil
	}

	var res interface{}
	if err := json.Unmarshal(val, &res); err != nil {
		return nil
	}

	switch v := res.(type) {
	case float64:
		return int64(v)
	default:
		return v
	}
}

func (r *RedisState) ClearData(userID int64) {
	ctx := context.Background()
	redisKey := r.makeDataKey(userID)
	r.client.Del(ctx, redisKey)
}
