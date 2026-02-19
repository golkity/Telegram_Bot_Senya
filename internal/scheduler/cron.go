package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Job func(ctx context.Context) error

type taskType int

const (
	typeDaily taskType = iota
	typeWeekly
)

type task struct {
	name    string
	typ     taskType
	weekday time.Weekday
	hour    int
	minute  int
	job     Job
	lastRun time.Time
}

type Scheduler struct {
	tasks []task
	log   *slog.Logger
	wg    sync.WaitGroup
}

func New(log *slog.Logger) *Scheduler {
	return &Scheduler{
		tasks: make([]task, 0),
		log:   log,
	}
}

func (s *Scheduler) AddDailyTask(name string, hour, minute int, job Job) {
	s.tasks = append(s.tasks, task{
		name:   name,
		typ:    typeDaily,
		hour:   hour,
		minute: minute,
		job:    job,
	})
	s.log.Info("scheduled daily task", "name", name, "time", fmt.Sprintf("%02d:%02d", hour, minute))
}

func (s *Scheduler) AddWeeklyTask(name string, weekday time.Weekday, hour, minute int, job Job) {
	s.tasks = append(s.tasks, task{
		name:    name,
		typ:     typeWeekly,
		weekday: weekday,
		hour:    hour,
		minute:  minute,
		job:     job,
	})
	s.log.Info("scheduled weekly task", "name", name, "day", weekday.String(), "time", fmt.Sprintf("%02d:%02d", hour, minute))
}

func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info("scheduler started")

	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	time.Sleep(time.Until(nextMinute))

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		defer ticker.Stop()

		s.checkAndRun(ctx, time.Now())

		for {
			select {
			case <-ctx.Done():
				s.log.Info("scheduler stopping...")
				s.wg.Wait()
				s.log.Info("scheduler stopped")
				return
			case t := <-ticker.C:
				s.checkAndRun(ctx, t)
			}
		}
	}()
}

func (s *Scheduler) checkAndRun(ctx context.Context, t time.Time) {
	currentDay := t.Weekday()
	currentHour := t.Hour()
	currentMinute := t.Minute()

	for i := range s.tasks {
		tsk := &s.tasks[i]

		if tsk.lastRun.Format("2006-01-02 15:04") == t.Format("2006-01-02 15:04") {
			continue
		}

		shouldRun := false

		if tsk.typ == typeDaily {
			if currentHour == tsk.hour && currentMinute == tsk.minute {
				shouldRun = true
			}
		} else if tsk.typ == typeWeekly {
			if currentDay == tsk.weekday && currentHour == tsk.hour && currentMinute == tsk.minute {
				shouldRun = true
			}
		}

		if shouldRun {
			tsk.lastRun = t
			s.wg.Add(1)

			go func(currentTask *task) {
				defer s.wg.Done()
				defer func() {
					if r := recover(); r != nil {
						s.log.Error("panic in scheduled task", "task", currentTask.name, "panic", r)
					}
				}()

				s.log.Info("executing scheduled task", "name", currentTask.name)
				start := time.Now()

				if err := currentTask.job(ctx); err != nil {
					s.log.Error("scheduled task failed", "name", currentTask.name, "error", err)
				} else {
					s.log.Info("scheduled task completed", "name", currentTask.name, "duration", time.Since(start))
				}
			}(tsk)
		}
	}
}
