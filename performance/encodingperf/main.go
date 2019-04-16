/*
 * Copyright 2012-2019 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
