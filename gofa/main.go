package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/anton2920/gofa/database"
	"github.com/anton2920/gofa/event"
	"github.com/anton2920/gofa/log"
	"github.com/anton2920/gofa/net/html"
	"github.com/anton2920/gofa/net/http"
	"github.com/anton2920/gofa/net/http/http1"
	"github.com/anton2920/gofa/net/tcp"
	"github.com/anton2920/gofa/syscall"
	"github.com/anton2920/gofa/time"
)

type Fortune struct {
	ID      database.ID
	Message string

	Data [128]byte
}

const PageSize = 4096

var DateBufferPtr unsafe.Pointer

var FortunesDB *database.DB

func CreateFortune(fortune *Fortune) error {
	var fortuneDB Fortune
	var err error

	data := unsafe.Slice(&fortuneDB.Data[0], len(fortuneDB.Data))

	fortune.ID, err = database.IncrementNextID(FortunesDB)
	if err != nil {
		return fmt.Errorf("failed to increment fortune ID: %w", err)
	}

	fortuneDB.ID = fortune.ID
	database.String2DBString(&fortuneDB.Message, fortune.Message, data, 0)

	return database.Write(FortunesDB, fortuneDB.ID, &fortuneDB)
}

func GetFortunes(pos *int64, fortunes []Fortune) (int, error) {
	n, err := database.ReadMany(FortunesDB, pos, fortunes)
	if err != nil {
		return 0, err
	}

	for i := 0; i < n; i++ {
		fortune := &fortunes[i]
		fortune.Message = database.Offset2String(fortune.Message, &fortune.Data[0])
	}
	return n, nil
}

func FortunesHandler(w *http.Response, r *http.Request) error {
	fortunes := make([]Fortune, 12, 13)
	var pos int64

	_, err := GetFortunes(&pos, fortunes)
	if err != nil {
		http.ServerError(err)
	}
	fortunes = append(fortunes, Fortune{ID: database.ID(len(fortunes)), Message: "Additional fortune added at request time."})

	for i := 1; i < len(fortunes); i++ {
		for j := 0; j < i; j++ {
			if fortunes[i].Message < fortunes[j].Message {
				fortunes[i].ID, fortunes[j].ID = fortunes[j].ID, fortunes[i].ID
				fortunes[i].Message, fortunes[j].Message = fortunes[j].Message, fortunes[i].Message
			}
		}
	}

	w.Headers.Set("Content-Type", `text/html; charset="UTF-8"`)
	w.WriteString(html.Header)

	w.WriteString(`<head>`)
	{
		w.WriteString(`<title>`)
		w.WriteString("Fortunes")
		w.WriteString(`</title>`)
	}
	w.WriteString(`</head>`)

	w.WriteString(`<body>`)
	{
		w.WriteString(`<table>`)
		w.WriteString(`<tr>`)
		w.WriteString(`<th>`)
		w.WriteString("id")
		w.WriteString(`</th>`)
		w.WriteString(`<th>`)
		w.WriteString("message")
		w.WriteString(`</th>`)
		w.WriteString(`</tr>`)
		{
			for i := 0; i < len(fortunes); i++ {
				fortune := &fortunes[i]

				w.WriteString(`<tr>`)

				w.WriteString(`<td>`)
				w.WriteID(fortune.ID)
				w.WriteString(`</td>`)

				w.WriteString(`<td>`)
				w.WriteHTMLString(fortune.Message)
				w.WriteString(`</td>`)

				w.WriteString(`</tr>`)
			}
		}
		w.WriteString(`</table>`)
	}
	w.WriteString(`</body>`)

	w.WriteString(`</html>`)
	return nil
}

func Router(ctx *http.Context, ws []http.Response, rs []http.Request) {
	for i := 0; i < len(rs); i++ {
		w := &ws[i]
		r := &rs[i]

		if r.URL.Path == "/plaintext" {
			w.WriteString("Hello, world!\n")
		} else if r.URL.Path == "/fortunes" {
			if err := FortunesHandler(w, r); err != nil {
				httpError, ok := err.(http.Error)
				if ok {
					w.StatusCode = httpError.StatusCode
					w.WriteString(httpError.DisplayMessage)
				} else {
					w.StatusCode = http.StatusInternalServerError
					w.WriteString(err.Error())
				}
			}
		}
	}
}

func GetDateHeader() []byte {
	return unsafe.Slice((*byte)(atomic.LoadPointer(&DateBufferPtr)), time.RFC822Len)
}

func UpdateDateHeader(now int) {
	buffer := make([]byte, time.RFC822Len)
	time.PutTmRFC822(buffer, time.ToTm(now))
	atomic.StorePointer(&DateBufferPtr, unsafe.Pointer(&buffer[0]))
}

func ServerWorker(q *event.Queue) {
	events := make([]event.Event, 64)

	const batchSize = 32
	ws := make([]http.Response, batchSize)
	rs := make([]http.Request, batchSize)

	for {
		n, err := q.GetEvents(events)
		if err != nil {
			log.Errorf("Failed to get events from client queue: %v", err)
			continue
		}
		dateBuffer := GetDateHeader()

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
							http1.FillError(ctx, err, dateBuffer)
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
							http1.FillError(ctx, err, dateBuffer)
							http.CloseAfterWrite(ctx)
							break
						}
						Router(ctx, ws[:n], rs[:n])
						http1.FillResponses(ctx, ws[:n], dateBuffer)
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

func CreateFortunes() error {
	fortunes := [...]Fortune{
		{Message: `fortune: No such file or directory`},
		{Message: `A computer scientist is someone who fixes things that aren't broken.`},
		{Message: `After enough decimal places, nobody gives a damn.`},
		{Message: `A bad random number generator: 1, 1, 1, 1, 1, 4.33e+67, 1, 1, 1`},
		{Message: `A computer program does what you tell it to do, not what you want it to do.`},
		{Message: `Emacs is a nice operating system, but I prefer UNIX. — Tom Christaensen`},
		{Message: `Any program that runs right is obsolete.`},
		{Message: `A list is only as strong as its weakest link. — Donald Knuth`},
		{Message: `Feature: A bug with seniority.`},
		{Message: `Computers make very fast, very accurate mistakes.`},
		{Message: `<script>alert("This should not be displayed in a browser alert box.");</script>`},
		{Message: `フレームワークのベンチマーク`},
	}

	if err := database.Drop(FortunesDB); err != nil {
		return fmt.Errorf("failed to drop fortunes data: %w", err)
	}
	for i := 0; i < len(fortunes); i++ {
		if err := CreateFortune(&fortunes[i]); err != nil {
			return fmt.Errorf("failed to create fortune %d: %w", err)
		}
	}

	return nil
}

func main() {
	var err error

	FortunesDB, err = database.Open("Fortunes.db")
	if err != nil {
		log.Fatalf("Failed to open fortunes DB file: %v", err)
	}
	defer database.Close(FortunesDB)

	if err := CreateFortunes(); err != nil {
		log.Fatalf("Failed to create fortunes: %v", err)
	}

	const address = "0.0.0.0:7073"
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
	now := time.Unix()
	UpdateDateHeader(now)

	events := make([]event.Event, 64)
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
				ctx, err := http.Accept(l, 1024)
				if err != nil {
					log.Errorf("Failed to accept new HTTP connection: %v", err)
					continue
				}
				_ = qs[counter%len(qs)].AddHTTP(ctx, event.RequestRead, event.TriggerEdge)
				counter++
			case event.Timer:
				now += e.Data
				UpdateDateHeader(now)
			case event.Signal:
				log.Infof("Received signal %d, exitting...", e.Identifier)
				quit = true
				break
			}
		}
	}
}
