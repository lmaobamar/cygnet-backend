package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	IdleTimeout   = 10 * 24 * time.Hour // each use pushes expiry out another 10 days
	touchInterval = 5 * time.Minute     // don't write to Redis on every request
)

var ErrNotFound = errors.New("session not found")

type Info struct {
	ID        string // hash of the token
	UserID    uuid.UUID
	CreatedAt time.Time
	LastSeen  time.Time
	IP        string
	UserAgent string
}

type Store struct{ rdb *redis.Client }

func NewStore(rdb *redis.Client) *Store { return &Store{rdb: rdb} }

func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sessionKey(tokenHash string) string { return "session:" + tokenHash }
func userKey(id uuid.UUID) string        { return "user_sessions:" + id.String() }

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, ip, userAgent string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	tokenHash := hash(token)
	now := time.Now().Unix()

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, sessionKey(tokenHash),
		"user_id", userID.String(),
		"created_at", now,
		"last_seen", now,
		"ip", clip(ip, 64),
		"ua", clip(userAgent, 200),
	)
	pipe.Expire(ctx, sessionKey(tokenHash), IdleTimeout)
	pipe.SAdd(ctx, userKey(userID), tokenHash)
	pipe.Expire(ctx, userKey(userID), IdleTimeout)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) load(ctx context.Context, tokenHash string) (*Info, error) {
	fields, err := s.rdb.HGetAll(ctx, sessionKey(tokenHash)).Result()
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, ErrNotFound
	}
	userID, err := uuid.Parse(fields["user_id"])
	if err != nil {
		return nil, ErrNotFound
	}
	created, _ := strconv.ParseInt(fields["created_at"], 10, 64)
	seen, _ := strconv.ParseInt(fields["last_seen"], 10, 64)
	return &Info{
		ID:        tokenHash,
		UserID:    userID,
		CreatedAt: time.Unix(created, 0),
		LastSeen:  time.Unix(seen, 0),
		IP:        fields["ip"],
		UserAgent: fields["ua"],
	}, nil
}

// Get validates a cookie token and slides the idle timeout forward
func (s *Store) Get(ctx context.Context, token string) (*Info, error) {
	tokenHash := hash(token)
	info, err := s.load(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if now.Sub(info.LastSeen) > touchInterval {
		pipe := s.rdb.Pipeline()
		pipe.HSet(ctx, sessionKey(tokenHash), "last_seen", now.Unix())
		pipe.Expire(ctx, sessionKey(tokenHash), IdleTimeout)
		pipe.Expire(ctx, userKey(info.UserID), IdleTimeout)
		_, _ = pipe.Exec(ctx) // best effort: failing to extend isn't worth failing the request
		info.LastSeen = now
	}
	return info, nil
}

func (s *Store) deleteByHash(ctx context.Context, userID uuid.UUID, tokenHash string) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(tokenHash))
	pipe.SRem(ctx, userKey(userID), tokenHash)
	_, err := pipe.Exec(ctx)
	return err
}

// Delete revokes the session behind a cookie token (logout)
func (s *Store) Delete(ctx context.Context, token string) error {
	tokenHash := hash(token)
	info, err := s.load(ctx, tokenHash)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.deleteByHash(ctx, info.UserID, tokenHash)
}

// DeleteByID revokes one of a user's sessions by its public ID. It only works on the user's own
func (s *Store) DeleteByID(ctx context.Context, userID uuid.UUID, id string) error {
	owned, err := s.rdb.SIsMember(ctx, userKey(userID), id).Result()
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return s.deleteByHash(ctx, userID, id)
}

// DeleteAll revokes every session a user has
func (s *Store) DeleteAll(ctx context.Context, userID uuid.UUID) error {
	hashes, err := s.rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	for _, tokenHash := range hashes {
		pipe.Del(ctx, sessionKey(tokenHash))
	}
	pipe.Del(ctx, userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

// List returns a user's live sessions, most recently used first
func (s *Store) List(ctx context.Context, userID uuid.UUID) ([]Info, error) {
	hashes, err := s.rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Info, 0, len(hashes))
	for _, tokenHash := range hashes {
		info, err := s.load(ctx, tokenHash)
		if errors.Is(err, ErrNotFound) {
			s.rdb.SRem(ctx, userKey(userID), tokenHash) // expired: tidy the index
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, *info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out, nil
}
