// Copyright 2012-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"github.com/nats-io/nats-mq/message"
	"log"
	"runtime"
	"time"

	"github.com/nats-io/go-nats"
)

func usage() {
	log.Printf("Usage: tock [-s server]\n")
	flag.PrintDefaults()
}

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 0 {
		usage()
		return
	}

	opts := []nats.Option{nats.Name("MQ-NATS Bridge tick-tock (ticker) example")}
	opts = setupConnOptions(opts)

	nc, err := nats.Connect(*urls, opts...)
	if err != nil {
		log.Fatal(err)
	}

	subj := "tock"

	nc.Subscribe(subj, func(msg *nats.Msg) {
		bridgeMsg, err := message.DecodeBridgeMessage(msg.Data)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Received message:\n")
		fmt.Printf("\tbody: %s\n", string(bridgeMsg.Body))

		counter, ok := bridgeMsg.GetInt64Property("counter")
		if !ok {
			log.Fatal("counter property is missing")
		}
		fmt.Printf("\tcounter: %d\n", counter)

		time, ok := bridgeMsg.GetStringProperty("time")
		if !ok {
			log.Fatal("time property is missing")
		}
		fmt.Printf("\ttime: %s\n", time)

		fmt.Printf("\tid: %s\n", string(bridgeMsg.Header.CorrelID))
		fmt.Println()
	})
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening on tock...")
	fmt.Println()
	runtime.Goexit()
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Printf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatal("Exiting, no servers available")
	}))
	return opts
}
