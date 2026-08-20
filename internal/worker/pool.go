package worker

import (
	"context"
	"log/slog"
	"sync"
)

// Job adalah fungsi yang akan dieksekusi di background
type Job func(ctx context.Context) error

type Pool interface {
	Start(ctx context.Context)
	Stop()
	Enqueue(job Job)
}

type pool struct {
	jobs    chan Job
	wg      sync.WaitGroup
	workers int
}

// NewPool menginisiasi Worker Pool dengan jumlah pekerja dan batas antrean
func NewPool(workers int, queueSize int) Pool {
	return &pool{
		jobs:    make(chan Job, queueSize),
		workers: workers,
	}
}

func (p *pool) Start(ctx context.Context) {
	for i := 1; i <= p.workers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			slog.Info("Worker background siap", slog.Int("worker_id", workerID))

			for job := range p.jobs {
				// Eksekusi tugas menggunakan context global dari aplikasi,
				// BUKAN context dari HTTP request (yang bisa expired kapan saja)
				if err := job(ctx); err != nil {
					slog.Error("Tugas background gagal", slog.Int("worker_id", workerID), slog.String("error", err.Error()))
				}
			}
		}(i)
	}
}

// Stop menghentikan penerimaan tugas baru dan menunggu tugas yang sedang berjalan selesai
func (p *pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	slog.Info("Semua worker background telah dimatikan secara aman")
}

func (p *pool) Enqueue(job Job) {
	p.jobs <- job
}
