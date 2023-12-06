package main

import (
	"encoding/json"
	"fmt"
	"sync"

	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/tuliofergulha/stock-exchange/go/internal/infra/kafka"
	"github.com/tuliofergulha/stock-exchange/go/internal/market/dto"
	"github.com/tuliofergulha/stock-exchange/go/internal/market/entity"
	"github.com/tuliofergulha/stock-exchange/go/internal/market/transformer"
)

func main() {
	ordersIn := make(chan *entity.Order)
	ordersOut := make(chan *entity.Order)
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	kafkaMsgChan := make(chan *ckafka.Message)
	configMap := &ckafka.ConfigMap{
		"bootstrap.servers": "host.docker.internal:9094",
		"groud.id":          "myGroup",
		"auto.offset.reset": "earliest",
	}

	producer := kafka.NewKafkaProducer(configMap)
	kafka := kafka.NewKafkaConsumer(configMap, []string{"input"})

	go kafka.Consume(kafkaMsgChan) // "go" creates a new thread

	book := entity.NewBook(ordersIn, ordersOut, wg)

	go book.Trade() // "go" creates a new thread

	go func() {
		for msg := range kafkaMsgChan {
			wg.Add(1)
			fmt.Println(string(msg.Value))
			tradeInput := dto.TradeInput{}
			err := json.Unmarshal(msg.Value, &tradeInput)
			if err != nil {
				panic(err)
			}
			order := transformer.TransformInput(tradeInput)
			ordersIn <- order
		}
	}()

	for res := range ordersOut {
		output := transformer.TransformOutput(res)
		outputJson, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		producer.Publish(outputJson, []byte("orders"), "output")
	}
}
