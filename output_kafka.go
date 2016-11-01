package main

import (
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/buger/gor/proto"
	"io"
	"log"
	"strings"
	"time"
)

// KafkaConfig should contains required information to
// build producers.
type KafkaConfig struct {
	host  string
	topic string
}

// KafkaOutput should make producer client.
type KafkaOutput struct {
	config   *KafkaConfig
	producer sarama.AsyncProducer
}

// KafkaMessage should contains catched request information that should be
// passed as Json to Apache Kafka.
type KafkaMessage struct {
	ReqURL             string `json:"Req_URL"`
	ReqMethod          string `json:"Req_Method"`
	ReqUserAgent       string `json:"Req_User-Agent"`
	ReqAcceptLanguage  string `json:"Req_Accept-Language,omitempty"`
	ReqAccept          string `json:"Req_Accept,omitempty"`
	ReqAcceptEncoding  string `json:"Req_Accept-Encoding,omitempty"`
	ReqIfModifiedSince string `json:"Req_If-Modified-Since,omitempty"`
	ReqConnection      string `json:"Req_Connection,omitempty"`
	ReqCookies         string `json:"Req_Cookies,omitempty"`
}

// KafkaOutputFrequency in milliseconds
const KafkaOutputFrequency = 500

// NewKafkaOutput creates instance of kafka producer client.
func NewKafkaOutput(address string, config *KafkaConfig) io.Writer {
	c := sarama.NewConfig()
	c.Producer.RequiredAcks = sarama.WaitForLocal
	c.Producer.Compression = sarama.CompressionSnappy
	c.Producer.Flush.Frequency = KafkaOutputFrequency * time.Millisecond

	brokerList := strings.Split(config.host, ",")

	producer, err := sarama.NewAsyncProducer(brokerList, c)
	if err != nil {
		log.Fatalln("Failed to start Sarama(Kafka) producer:", err)
	}

	o := &KafkaOutput{
		config:   config,
		producer: producer,
	}

	if Settings.verbose {
		// Start infinite loop for tracking errors for kafka producer.
		go o.ErrorHandler()
	}

	return o
}

// ErrorHandler should receive errors
func (o *KafkaOutput) ErrorHandler() {
	for err := range o.producer.Errors() {
		log.Println("Failed to write access log entry:", err)
	}
}

func (o *KafkaOutput) Write(data []byte) (n int, err error) {
	kafkaMessage := KafkaMessage{
		ReqURL:             string(proto.Path(data)),
		ReqMethod:          string(proto.Method(data)),
		ReqUserAgent:       string(proto.Header(data, []byte("User-Agent"))),
		ReqAcceptLanguage:  string(proto.Header(data, []byte("Accept-Language"))),
		ReqAccept:          string(proto.Header(data, []byte("Accept"))),
		ReqAcceptEncoding:  string(proto.Header(data, []byte("Accept-Encoding"))),
		ReqIfModifiedSince: string(proto.Header(data, []byte("If-Modified-Since"))),
		ReqConnection:      string(proto.Header(data, []byte("Connection"))),
		ReqCookies:         string(proto.Header(data, []byte("Cookie"))),
	}
	jsonMessage, _ := json.Marshal(&kafkaMessage)
	message := sarama.StringEncoder(jsonMessage)

	o.producer.Input() <- &sarama.ProducerMessage{
		Topic: o.config.topic,
		Value: message,
	}

	return len(message), nil
}
