package cms

import (
	"context"
	"log/slog"
	"sync"
)

type Service struct {
	repo  Repository
	log   *slog.Logger
	cache map[string]string
	mu    sync.RWMutex
}

func NewService(repo Repository, log *slog.Logger) *Service {
	s := &Service{
		repo:  repo,
		log:   log,
		cache: make(map[string]string),
	}
	_ = s.LoadCache(context.Background())
	return s
}

func (s *Service) LoadCache(ctx context.Context) error {
	contents, err := s.repo.GetAll(ctx)
	if err != nil {
		s.log.Error("failed to load cms cache", "error", err)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range contents {
		s.cache[c.Key] = c.Value
	}
	s.log.Info("cms cache loaded", "count", len(s.cache))
	return nil
}

func (s *Service) GetText(ctx context.Context, key string, defaultVal string) string {
	s.mu.RLock()
	val, ok := s.cache[key]
	s.mu.RUnlock()

	if ok {
		return val
	}

	dbVal, err := s.repo.GetByKey(ctx, key)
	if err == nil {
		s.mu.Lock()
		s.cache[key] = dbVal
		s.mu.Unlock()
		return dbVal
	}

	_ = s.repo.Update(ctx, key, defaultVal)
	s.mu.Lock()
	s.cache[key] = defaultVal
	s.mu.Unlock()

	return defaultVal
}

func (s *Service) UpdateText(ctx context.Context, key string, value string) error {
	err := s.repo.Update(ctx, key, value)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()

	return nil
}

func (s *Service) GetAllContent(ctx context.Context) ([]Content, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) UpdateSetting(ctx context.Context, key string, content string) error {
	return s.repo.UpdateSetting(ctx, key, content)
}
