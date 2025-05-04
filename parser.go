package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"sync"
	"time"
)

var urlRe = regexp.MustCompile(`https?://[^\s"']+`)

func extractURLs(data []byte) []string {
	// Note: converts to string; for very large data you might
	// scan in chunks instead of one big string.
	return urlRe.FindAllString(string(data), -1)
}

type (
	Parser struct {
		hc   *http.Client
		done chan struct{}

		db    db
		cache cache

		records []record
		mx      sync.RWMutex

		numWorkers int
	}

	db interface {
		Insert(ctx context.Context, rec []record) error
		List(ctx context.Context) ([]record, error)
	}

	cache interface {
		IsUsed(ctx context.Context, key string) bool
		Set(ctx context.Context, key string)
	}

	tmpRec struct {
		parent string
		curr   string
	}
)

func NewParser(db db, cache cache) *Parser {
	return &Parser{
		hc: &http.Client{
			Timeout: 20 * time.Second,
		},

		db:    db,
		cache: cache,

		done: make(chan struct{}),

		records: make([]record, 0, 1000),

		numWorkers: runtime.GOMAXPROCS(0),
	}
}

func (p *Parser) Walk(ctx context.Context, entryPoint string) {

	if err := p.parse(ctx, entryPoint); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Println(err)
		return
	}

	fmt.Println("All good")
}

func (p *Parser) Close() error {
	close(p.done)
	return nil
}

func (p *Parser) parse(ctx context.Context, entryPoint string) error {

	t := time.NewTicker(150 * time.Millisecond)

	ch := make(chan tmpRec, p.numWorkers)
	ch <- tmpRec{parent: "", curr: entryPoint}
	for {
		select {
		case <-p.done:
			return nil

		case <-ctx.Done():
			fmt.Println("Finished by context cancel")
			return ctx.Err()

		case <-t.C:
			go p.sendRequest(ctx, ch)
		}
	}
}

func (p *Parser) sendRequest(ctx context.Context, ch chan tmpRec) {
	curr := <-ch
	if !p.cache.IsUsed(ctx, curr.curr) {
		fmt.Printf("Already used link: %s\n", curr.curr)
		return
	}

	resp, err := http.Get(curr.curr)
	if err != nil {
		p.cache.Set(ctx, curr.curr)
		fmt.Printf("Failed to send request: %s\n", curr.curr)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.cache.Set(ctx, curr.curr)
		fmt.Printf("Failed to read body: %s\n", curr.curr)
		return
	}

	urls := extractURLs(body)
	for _, url := range urls {
		ch <- tmpRec{parent: curr.curr, curr: url}
	}

	p.mx.Lock()

	p.records = append(p.records, record{
		Parent:    curr.parent,
		Child:     curr.curr,
		Content:   body,
		CreatedAt: time.Now(),
	})

	if len(p.records) == 100 {
		if err := p.db.Insert(ctx, p.records); err != nil {
			fmt.Printf("Failed to insert records: %s\n", curr.parent)
		}
		fmt.Println("Successfully inserted records")
		p.records = p.records[:0]
	}

	p.mx.Unlock()
}
