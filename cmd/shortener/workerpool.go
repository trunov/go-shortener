package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/trunov/go-shortener/internal/app/handler"
)

type Workerpool struct {
	storage handler.Storager
}

type Job interface {
	Run(ctx context.Context) error
}

type DeleteURLSJob struct {
	storage     handler.Storager
	shortenURLS []string
	userID      string
}

func NewWorkerpool(storage *handler.Storager) *Workerpool {
	return &Workerpool{storage: *storage}
}

func (j *DeleteURLSJob) Run(ctx context.Context) error {
	fmt.Println("job has started")
	err := j.storage.DeleteURLS(ctx, j.userID, j.shortenURLS)
	if err != nil {
		return err
	}
	return nil
}

func (w *Workerpool) runPool(ctx context.Context, jobs chan Job) error {
	gr, ctx := errgroup.WithContext(ctx)
	for i := 0; i < runtime.GOMAXPROCS(runtime.NumCPU()-1); i++ {
		gr.Go(func() error {
			for {
				select {
				case job := <-jobs:
					err := job.Run(ctx)
					if err != nil {
						return err
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
	jobs := make(chan Job)

	go func() {
		for inputCh != nil {
			v, ok := <-inputCh
			if !ok {
				inputCh = nil
				continue
			}
			jobs <- &DeleteURLSJob{
				storage:     w.storage,
				shortenURLS: v,
				userID:      userID,
			}

		}
	}()

	defer func() {
		close(jobs)
	}()

	err := w.runPool(ctx, jobs)
	if err != nil {
		log.Println(err)
	}
}
