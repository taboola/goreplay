package replay

import (
	"log"
	"time"
)

func RunReplayFromFile(rf *RequestFactory) {
	TotalResponsesCount = 0

	log.Println("Starting file reply")
	requests, err := parseReplyFile()

	if err != nil {
		log.Fatal("Can't parse request: ", err)
	}

	var lastTimestamp int64

	if len(requests) > 0 {
		lastTimestamp = requests[0].Timestamp
	}

	requestsToReplay := 0

	hosts := Settings.ForwardedHosts()
	for _, host := range hosts {
		if host.Limit > 0 {
			requestsToReplay += host.Limit
		} else {
			requestsToReplay += len(requests)
		}
	}

	for _, request := range requests {

		parsedReq, err := ParseRequest(request.Request)

		if err != nil {
			log.Fatal("Can't parse request...:", err)
		}

		time.Sleep(time.Duration(request.Timestamp - lastTimestamp))

		rf.Add(parsedReq)

		lastTimestamp = request.Timestamp
	}

	for requestsToReplay > TotalResponsesCount {
		time.Sleep(time.Second)
	}

}
