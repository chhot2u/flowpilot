package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	DefaultPoolSize   = 5
	MaxPoolSize       = 50
	PoolIdleTimeout   = 5 * time.Minute
	PoolDialTimeout   = 30 * time.Second
	PoolCleanupPeriod = 30 * time.Second
)

type pooledBrowser struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	lastUsed    time.Time
	inUse       int
	maxTabs     int
}

type BrowserPool struct {
	mu          sync.Mutex
	browsers    []*pooledBrowser
	poolSize    int
	maxTabs     int
	idleTimeout time.Duration
	opts        []chromedp.ExecAllocatorOption
	stopped     bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

type PoolConfig struct {
	Size        int
	MaxTabs     int
	IdleTimeout time.Duration
}

func NewBrowserPool(cfg PoolConfig, opts []chromedp.ExecAllocatorOption) *BrowserPool {
	if cfg.Size <= 0 {
		cfg.Size = DefaultPoolSize
	}
	if cfg.Size > MaxPoolSize {
		cfg.Size = MaxPoolSize
	}
	if cfg.MaxTabs <= 0 {
		cfg.MaxTabs = 10
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = PoolIdleTimeout
	}

	p := &BrowserPool{
		browsers:    make([]*pooledBrowser, 0, cfg.Size),
		poolSize:    cfg.Size,
		maxTabs:     cfg.MaxTabs,
		idleTimeout: cfg.IdleTimeout,
		opts:        opts,
		stopCh:      make(chan struct{}),
	}

	p.wg.Add(1)
	go p.cleanupLoop()

	return p
}

func (p *BrowserPool) Acquire(ctx context.Context) (browserCtx context.Context, release func(), err error) {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("browser pool is stopped")
	}

	for _, b := range p.browsers {
		if b.inUse < b.maxTabs {
			b.inUse++
			b.lastUsed = time.Now()
			allocCtx := b.allocCtx
			p.mu.Unlock()

			tabCtx, tabCancel := chromedp.NewContext(allocCtx,
				chromedp.WithNewBrowserContext())
			release = func() {
				tabCancel()
				p.mu.Lock()
				b.inUse--
				b.lastUsed = time.Now()
				p.mu.Unlock()
			}
			return tabCtx, release, nil
		}
	}

	if len(p.browsers) < p.poolSize {
		b, err := p.createBrowser(ctx)
		if err != nil {
			p.mu.Unlock()
			return nil, nil, fmt.Errorf("create pooled browser: %w", err)
		}
		b.inUse++
		p.browsers = append(p.browsers, b)
		allocCtx := b.allocCtx
		p.mu.Unlock()

		tabCtx, tabCancel := chromedp.NewContext(allocCtx,
			chromedp.WithNewBrowserContext())
		release = func() {
			tabCancel()
			p.mu.Lock()
			b.inUse--
			b.lastUsed = time.Now()
			p.mu.Unlock()
		}
		return tabCtx, release, nil
	}

	p.mu.Unlock()
	return nil, nil, fmt.Errorf("browser pool exhausted: all %d browsers at max tab capacity", p.poolSize)
}

func (p *BrowserPool) createBrowser(ctx context.Context) (*pooledBrowser, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, p.opts...)

	browserCtx, _ := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(browserCtx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("warm up pooled browser: %w", err)
	}

	return &pooledBrowser{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		lastUsed:    time.Now(),
		maxTabs:     p.maxTabs,
	}, nil
}

func (p *BrowserPool) cleanupLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(PoolCleanupPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.evictIdle()
		}
	}
}

func (p *BrowserPool) evictIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	active := make([]*pooledBrowser, 0, len(p.browsers))
	for _, b := range p.browsers {
		if b.inUse == 0 && now.Sub(b.lastUsed) > p.idleTimeout {
			b.allocCancel()
		} else {
			active = append(active, b)
		}
	}
	p.browsers = active
}

func (p *BrowserPool) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true

	for _, b := range p.browsers {
		b.allocCancel()
	}
	p.browsers = nil
	p.mu.Unlock()

	close(p.stopCh)
	p.wg.Wait()
}

func (p *BrowserPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := PoolStats{
		TotalBrowsers: len(p.browsers),
		MaxBrowsers:   p.poolSize,
	}
	for _, b := range p.browsers {
		stats.ActiveTabs += b.inUse
		if b.inUse == 0 {
			stats.IdleBrowsers++
		}
	}
	return stats
}

type PoolStats struct {
	TotalBrowsers int `json:"totalBrowsers"`
	MaxBrowsers   int `json:"maxBrowsers"`
	ActiveTabs    int `json:"activeTabs"`
	IdleBrowsers  int `json:"idleBrowsers"`
}
