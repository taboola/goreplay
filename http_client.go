package main

import (
	"crypto/tls"
	"net"
	"net/url"
	"strings"
)

var defaultPorts = map[string]string{
	"http":  "80",
	"https": "443",
}

type HTTPClient struct {
	baseURL *url.URL
	conn    net.Conn
	buf     []byte
}

func NewHTTPClient(baseURL string) *HTTPClient {
	client := new(HTTPClient)
	client.baseURL, _ = url.Parse(baseURL)
	client.buf = make([]byte, 4096*10)

	if !strings.Contains(client.baseURL.Host, ":") {
		client.baseURL.Host += ":" + defaultPorts[client.baseURL.Scheme]
	}

	return client
}

func (c *HTTPClient) Connect() (err error) {
	c.conn, err = net.Dial("tcp", c.baseURL.Host)

	if c.baseURL.Scheme == "https" {
		tlsConn := tls.Client(c.conn, &tls.Config{InsecureSkipVerify: true})
		err = tlsConn.Handshake()
		c.conn = tlsConn
	}

	return
}

func (c *HTTPClient) Disconnect() {
	c.conn.Close()
	c.conn = nil
}

func (c *HTTPClient) Send(data []byte) (response []byte, err error) {
	if c.conn == nil {
		c.Connect()
	}

	_, err = c.conn.Write(data)
	n, err := c.conn.Read(c.buf)

	Debug(string(c.buf[:n]))

	return c.buf[:n], err
}
