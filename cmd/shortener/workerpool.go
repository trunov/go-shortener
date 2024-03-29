package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/trunov/go-shortener/internal/app/handler"
)

type Job interface {
	Run(ctx context.Context) error
}

type Workerpool struct {
	storage handler.Storager
	jobs    chan Job
	wg      sync.WaitGroup
}

type DeleteURLSJob struct {
	storage     handler.Storager
	shortenURLS []string
	userID      string
}

func NewWorkerpool(storage *handler.Storager) *Workerpool {
	wp := &Workerpool{
		storage: *storage,
		jobs:    make(chan Job, 10),
	}

	go wp.runPool(context.Background())

	return wp
}

func (j *DeleteURLSJob) Run(ctx context.Context) error {
	fmt.Println("job has started")
	err := j.storage.DeleteURLS(ctx, j.userID, j.shortenURLS)
	if err != nil {
		return err
	}
	return nil
}

func (w *Workerpool) runPool(ctx context.Context) error {
	gr, ctx := errgroup.WithContext(ctx)

	for i := 0; i < runtime.GOMAXPROCS(runtime.NumCPU()-1); i++ {
		w.wg.Add(1)
		gr.Go(func() error {
			defer w.wg.Done()
			for {
				select {
				case job, ok := <-w.jobs:
					if !ok {
						return nil
					}
					if err := job.Run(ctx); err != nil {
						log.Println(err)
					}

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}
	return gr.Wait()
}

func (w *Workerpool) Start(ctx context.Context, inputCh chan []string, userID string) {
	for shortenURLs := range inputCh {
		w.jobs <- &DeleteURLSJob{
			storage:     w.storage,
			shortenURLS: shortenURLs,
			userID:      userID,
		}
	}
}

func (w *Workerpool) Stop() {
	close(w.jobs)
	w.wg.Wait()
}
