package main

import (
	"flag"
	"fmt"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nuid"
	"log"
	"runtime"
	"time"

	"github.com/nats-io/go-nats"
)

func usage() {
	log.Printf("Usage: tick [-s server] <msg>\n")
	flag.PrintDefaults()
}

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		usage()
		return
	}

	opts := []nats.Option{nats.Name("MQ-NATS Bridge tick-tock (ticker) example")}

	nc, err := nats.Connect(*urls, opts...)
	if err != nil {
		log.Fatal(err)
	}

	subj := "tick"
	body := []byte(args[0])

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		counter := 0
		for t := range ticker.C {
			counter = counter + 1
			bridgeMsg := message.NewBridgeMessage(body)
			bridgeMsg.Header.CorrelID = []byte(nuid.Next())
			bridgeMsg.SetProperty("counter", counter)
			bridgeMsg.SetProperty("time", t.Format(time.UnixDate))

			fmt.Printf("Sending message:\n")
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

			encoded, err := bridgeMsg.Encode()

			if err != nil {
				log.Fatal(err)
			}
			nc.Publish(subj, encoded)
			nc.Flush()
		}
	}()

	fmt.Println("Running forever, use ctrl-c to cancel...")
	runtime.Goexit()
}
