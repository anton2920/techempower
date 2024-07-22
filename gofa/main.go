package main

import (
	"runtime"

	"github.com/anton2920/gofa/event"
	"github.com/anton2920/gofa/log"
	"github.com/anton2920/gofa/net/http"
	"github.com/anton2920/gofa/net/http/http1"
	"github.com/anton2920/gofa/net/tcp"
	"github.com/anton2920/gofa/syscall"
	"github.com/anton2920/gofa/time"
)

const PageSize = 4096

var DateBuffer = make([]byte, time.RFC822Len)

func Router(ctx *http.Context, ws []http.Response, rs []http.Request) {
	for i := 0; i < len(rs); i++ {
		w := &ws[i]
		r := &rs[i]

		if r.URL.Path == "/plaintext" {
			w.Write("Hello, world!\n")
		}
	}
}

func ServerWorker(q *event.Queue) {
	const batchSize = 64

	events := make([]event.Event, batchSize)
	ws := make([]http.Response, batchSize)
	rs := make([]http.Request, batchSize)

	for {
		n, err := q.GetEvents(events)
		if err != nil {
			log.Errorf("Failed to get events from client queue: %v", err)
			continue
		}

		for i := 0; i < n; i++ {
			e := &events[i]
			if errno := e.Error(); errno != 0 {
				log.Errorf("Event for %v returned code %d (%s)", e.Identifier, errno, errno)
				continue
			}

			ctx, ok := http.GetContextFromPointer(e.UserData)
			if !ok {
				continue
			}
			if e.EndOfFile() {
				http.Close(ctx)
				continue
			}

			switch e.Type {
			case event.Read:
				var read int
				for read < e.Data {
					n, err := http.Read(ctx)
					if err != nil {
						if err == http.NoSpaceLeft {
							http1.FillError(ctx, err, DateBuffer)
							http.CloseAfterWrite(ctx)
							break
						}
						log.Errorf("Failed to read data from client: %v", err)
						http.Close(ctx)
						break
					}
					read += n

					for n > 0 {
						n, err = http1.ParseRequestsUnsafe(ctx, rs)
						if err != nil {
							http1.FillError(ctx, err, DateBuffer)
							http.CloseAfterWrite(ctx)
							break
						}
						Router(ctx, ws[:n], rs[:n])
						http1.FillResponses(ctx, ws[:n], DateBuffer)
					}
				}
				fallthrough
			case event.Write:
				_, err = http.Write(ctx)
				if err != nil {
					log.Errorf("Failed to write data to client: %v", err)
					http.Close(ctx)
					continue
				}
			}
		}
	}
}

func main() {
	log.Infof("Starting gofa/benchmark...")

	const address = "0.0.0.0:7072"
	l, err := tcp.Listen(address, 128)
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	defer syscall.Close(l)

	log.Infof("Listening on %s...", address)

	q, err := event.NewQueue()
	if err != nil {
		log.Fatalf("Failed to create listener event queue: %v", err)
	}
	defer q.Close()

	_ = q.AddSocket(l, event.RequestRead, event.TriggerEdge, nil)
	_ = q.AddTimer(1, 1, event.Seconds, nil)

	_ = syscall.IgnoreSignals(syscall.SIGINT, syscall.SIGTERM)
	_ = q.AddSignals(syscall.SIGINT, syscall.SIGTERM)

	nworkers := min(runtime.GOMAXPROCS(0)/2, runtime.NumCPU())
	qs := make([]*event.Queue, nworkers)
	for i := 0; i < nworkers; i++ {
		qs[i], err = event.NewQueue()
		if err != nil {
			log.Fatalf("Failed to create new client queue: %v", err)
		}
		go ServerWorker(qs[i])
	}

	events := make([]event.Event, 64)
	now := time.Unix()
	var counter int

	var quit bool
	for !quit {
		n, err := q.GetEvents(events)
		if err != nil {
			log.Errorf("Failed to get events: %v", err)
			continue
		}

		for i := 0; i < n; i++ {
			e := &events[i]

			switch e.Type {
			default:
				log.Panicf("Unhandled event: %#v", e)
			case event.Read:
				ctx, err := http.Accept(l, PageSize)
				if err != nil {
					log.Errorf("Failed to accept new HTTP connection: %v", err)
					continue
				}
				_ = http.AddClientToQueue(qs[counter%len(qs)], ctx, event.RequestRead, event.TriggerEdge)
				counter++
			case event.Timer:
				now += e.Data
				time.PutTmRFC822(DateBuffer, time.ToTm(now))
			case event.Signal:
				log.Infof("Received signal %d, exitting...", e.Identifier)
				quit = true
				break
			}
		}
	}
}
