package worker

import (
	"context"
	"log/slog"
	"sync"
	"telegram_bot/internal/modules/report"
)

type Pool struct {
	tasks    chan report.Task
	provider DataProvider
	gen      ReportGenerator
	sender   FileSender
	log      *slog.Logger
	wg       *sync.WaitGroup
}

func NewPool(
	tasks chan report.Task,
	provider DataProvider,
	gen ReportGenerator,
	sender FileSender,
	log *slog.Logger,
) *Pool {
	return &Pool{
		tasks:    tasks,
		provider: provider,
		gen:      gen,
		sender:   sender,
		log:      log,
		wg:       &sync.WaitGroup{},
	}
}

func (p *Pool) Start(ctx context.Context, count int) {
	for i := 0; i < count; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			p.log.Info("worker started", "id", workerID)

			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					ProcessReportJob(ctx, task, p.provider, p.gen, p.sender, p.log)
				}
			}
		}(i)
	}
}

func (p *Pool) Stop() {
	close(p.tasks)
	p.wg.Wait()
}
