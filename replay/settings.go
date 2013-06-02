package replay

import (
	"strconv"
	"strings"
)

type ForwardHost struct {
	Url   string
	Limit int

	Stat *RequestStat
}

type ReplaySettings struct {
	port int
	host string

	forwardAddress string
}

// ForwardedHosts implements forwardAddress syntax support for multiple hosts (coma separated), and rate limiting by specifing "|maxRps" after host name.
//
//    -f "host1,http://host2|10,host3"
//
func (r *ReplaySettings) ForwardedHosts() (hosts []*ForwardHost) {
	hosts = make([]*ForwardHost, 0, 10)

	for _, address := range strings.Split(r.address, ",") {
		host_info := strings.Split(address, "|")

		if strings.Index(host_info[0], "http") == -1 {
			host_info[0] = "http://" + host_info[0]
		}

		host := &ForwardHost{Url: host_info[0]}
		host.Stat = NewRequestStats(host)

		if len(host_info) > 1 {
			host.Limit, _ = strconv.Atoi(host_info[1])
		}

		hosts = append(hosts, host)
	}

	return
}

// Helper to return address with port, e.g.: 127.0.0.1:28020
func (r *ReplaySettings) Address() string {
	return r.host + ":" + strconv.Itoa(r.port)
}
