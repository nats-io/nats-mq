package main

import (
	"flag"
	"log"
	"time"

	"github.com/nats-io/nats-mq/message"
)

var iterations int

func main() {
	flag.IntVar(&iterations, "i", 100, "iterations")
	flag.Parse()

	log.Printf("encoding and decoding %d messages", iterations)

	msg := message.NewBridgeMessage([]byte("hello"))
	msg.SetProperty("one", int32(249))
	toDecode, err := msg.Encode()
	if err != nil {
		log.Fatalf("error encoding message, %s", err.Error())
	}

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := msg.Encode()
		if err != nil {
			log.Fatalf("error encoding message, %s", err.Error())
		}
	}
	end := time.Now()

	diff := end.Sub(start)
	rate := float64(iterations) / float64(diff.Seconds())
	log.Printf("Encoded %d messages in %s, or %.2f msgs/sec", iterations, diff, rate)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		decoded, err := message.DecodeBridgeMessage(toDecode)
		if err != nil || string(decoded.Body) != "hello" {
			log.Fatalf("error decoding message, %s", err.Error())
		}
	}
	end = time.Now()

	diff = end.Sub(start)
	rate = float64(iterations) / float64(diff.Seconds())
	log.Printf("Decoded %d messages in %s, or %.2f msgs/sec", iterations, diff, rate)
}
