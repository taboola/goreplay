package main

import (
	"github.com/Shopify/sarama"
	"log"
	"strings"
	"time"
)

// KafkaConfig should contains required information to
// build producers.
type KafkaConfig struct {
	zookeeper string
	topic     string
}

// KafkaOutput should make producer client.
type KafkaOutput struct {
	address  string
	config   *KafkaConfig
	producer sarama.AsyncProducer
}

// NewKafkaOutput creates instance of kafka producer client.
func NewKafkaOutput(address string, config *KafkaConfig) *KafkaOutput {
	c := sarama.NewConfig()
	c.Producer.RequiredAcks = sarama.WaitForLocal
	c.Producer.Compression = sarama.CompressionSnappy
	c.Producer.Flush.Frequency = 500 * time.Millisecond

	brokerList := strings.Split(config.zookeeper, ",")

	producer, err := sarama.NewAsyncProducer(brokerList, c)
	if err != nil {
		log.Fatalln("Failed to start Sarama(Kafka) producer:", err)
	}

	o := &KafkaOutput{
		address:  address,
		config:   config,
		producer: producer,
	}

	// Start infinite loop for tracking errors for kafka producer.
	go o.ErrorHandler()

	return o
}

// ErrorHandler should receive errors
func (o *KafkaOutput) ErrorHandler() {
	for err := range o.producer.Errors() {
		log.Println("Failed to write access log entry:", err)
	}
}

func (o *KafkaOutput) Write(data []byte) (n int, err error) {
	buf := make(sarama.ByteEncoder, len(data))
	copy(buf, data)

	o.producer.Input() <- &sarama.ProducerMessage{
		Topic: o.config.topic,
		Key:   sarama.StringEncoder(o.address),
		Value: buf,
	}

	return len(data), nil
}
