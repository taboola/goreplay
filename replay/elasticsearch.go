package replay

import (
	"encoding/json"
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"log"
	"mime"
	"strconv"
	"strings"
	"time"
)

type ESPlugin struct {
	Active  bool
	ApiPort int
	Host    string
	Index   string
	indexor *core.BulkIndexor
	done    chan bool
}

type ESResponse struct {
	Url              string `json:"URL"`
	Status           string
	StatusCode       int
	Proto            string
	ContentLength    int64
	TransferEncoding []string
	MimeType         string
	Rtt              int64 `json:"RTT"`
	Timestamp        time.Time
}

func (p *ESPlugin) Init() {
	// Start the Handler go routine
	api.Domain = p.Host
	api.Port = strconv.Itoa(p.ApiPort)
	p.indexor = core.NewBulkIndexorErrors(50, 60)
	p.done = make(chan bool)
	p.indexor.Run(p.done)
	if Settings.Verbose {
		// Only start the ErrorHandler goroutine when in verbose mode
		// no need to burn ressources otherwise
		go p.ErrorHandler()
	}
	log.Println("Initialized Elasticsearch Plugin")
	return
}

func (p *ESPlugin) IndexerShutdown() {
	p.done <- true
	return
}

func (p *ESPlugin) ErrorHandler() {
	for {
		errBuf := <-p.indexor.ErrorChannel
		log.Println(errBuf.Err)
	}
}

func (p *ESPlugin) RttDurationToMs(d time.Duration) int64 {
	sec := d / time.Second
	nsec := d % time.Second
	fl := float64(sec) + float64(nsec)*1e-6
	return int64(fl)
}

func (p *ESPlugin) GetMimeFromUrl(url string) string {
	// get extension string
	split := strings.Split(url, "/")
	ls := split[len(split)-1]
	if strings.Contains(ls, ".") {
		// could be a file with extension
		extsplit := strings.Split(ls, ".")
		extls := extsplit[len(extsplit)-1]
		qstrsplit := strings.Split(extls, "?")
		m := mime.TypeByExtension("." + qstrsplit[0])
		if len(m) > 0 {
			Debug("MimeType: " + m)
		} else {
			Debug("MimeType: Not found")
		}
		return m
	} else {
		// no extension - no mime type
		return ""
	}
}

func (p *ESPlugin) ResponseAnalyze(r *HttpResponse) {
	t := time.Now()
	rtt := p.RttDurationToMs(r.timing.respDone.Sub(r.timing.reqStart))

	resp := ESResponse{
		Url:              r.req.URL.String(),
		Status:           r.resp.Status,
		StatusCode:       r.resp.StatusCode,
		Proto:            r.resp.Proto,
		ContentLength:    r.resp.ContentLength,
		TransferEncoding: r.resp.TransferEncoding,
		MimeType:         p.GetMimeFromUrl(r.req.URL.String()),
		Rtt:              rtt,
		Timestamp:        t,
	}
	j, err := json.Marshal(&resp)
	if err != nil {
		log.Println(err)
	} else {
		if Settings.Verbose {
			log.Printf("Elasticsearch - Response to Index: %s", j)
		}
		p.indexor.Index(p.Index, "response", "", "", &t, j)
	}
	return
}
