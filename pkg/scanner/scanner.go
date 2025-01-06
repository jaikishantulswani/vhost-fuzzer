package scanner
import (
	"fmt"
	"sync"
	"time"

	"github.com/dsecuredcom/vhost-fuzzer/pkg/config"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/time/rate" // Import the rate package
)

type Scanner struct {
	config         config.Config
	bar            *progressbar.ProgressBar
	targetChan     chan Target
	resultChan     chan string
	clients        *clientCache
	progressCount  int64
	progressMutex  sync.Mutex
	lastUpdateTime time.Time
	rateLimiter    *rate.Limiter // Add a rate limiter
}

func NewScanner(cfg config.Config, bar *progressbar.ProgressBar) *Scanner {
	var rateLimiter *rate.Limiter
	if cfg.RateLimit > 0 {
		rateLimiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), 1) // Allow cfg.RateLimit requests per second
	}

	return &Scanner{
		config:         cfg,
		bar:            bar,
		targetChan:     make(chan Target, cfg.Concurrency*2),
		resultChan:     make(chan string, cfg.Concurrency*2),
		clients:        newClientCache(),
		progressCount:  0,
		progressMutex:  sync.Mutex{},
		lastUpdateTime: time.Now(),
		rateLimiter:    rateLimiter, // Initialize the rate limiter
	}
}

func (s *Scanner) Run() {
	processor, err := NewBatchProcessor(
		s.config.IPsFile,
		s.config.HostsFile,
		s.config.Paths,
		s.targetChan,
	)
	if err != nil {
		fmt.Printf("Error initializing batch processor: %v\n", err)
		return
	}
	defer processor.Close()

	go func() {
		if err := processor.ProcessFilesChunked(); err != nil {
			fmt.Printf("Error processing files: %v\n", err)
		}
	}()

	pool := NewWorkerPool(s.config.Concurrency, s)
	pool.Start()

	done := make(chan struct{})
	go s.processResults(done)

	pool.Wait()
	close(s.resultChan)
	<-done
}

func (s *Scanner) updateProgress() {
	s.progressMutex.Lock()
	s.progressCount++

	if s.progressCount%progressBatch == 0 || time.Since(s.lastUpdateTime) > time.Second {
		s.bar.Set(int(s.progressCount))
		s.lastUpdateTime = time.Now()
	}
	s.progressMutex.Unlock()
}

func (s *Scanner) processResults(done chan struct{}) {
	for result := range s.resultChan {
		fmt.Println(result)
	}
	close(done)
}
