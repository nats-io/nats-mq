package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nats-mq/nats-mq/core"
)

var iterations int

func main() {
	flag.IntVar(&iterations, "i", 1000, "iterations, docker image defaults to 5000 in queue")
	flag.Parse()
	msg := strings.Repeat("stannats", 128) // 1024 bytes

	connect := []conf.ConnectorConfig{}

	for i := 1; i <= 10; i++ {
		connect = append(connect, conf.ConnectorConfig{
			Type:                  "Queue2NATS",
			Subject:               fmt.Sprintf("test.%d", i),
			Queue:                 fmt.Sprintf("TEST.QUEUE.%d", i),
			ExcludeHeaders:        true,
			MaxMQMessagesInFlight: 25,
		})
	}

	// Start the infrastructure
	tbs, err := core.StartTestEnvironmentInfrastructure(false)
	if err != nil {
		log.Fatalf("error starting test environment, %s", err.Error())
	}

	start := time.Now()
	done := make(chan bool)
	var count uint64
	maxCount := uint64(iterations * len(connect))

	for _, c := range connect {
		tbs.NC.Subscribe(c.Subject, func(msg *nats.Msg) {
			if atomic.LoadUint64(&count) == 0 {
				log.Printf("received first message")
				start = time.Now() // start on the first message
			}

			newCount := atomic.AddUint64(&count, 1)
			if newCount%5000 == 0 {
				log.Printf("received count = %d", count)
			}

			if newCount == maxCount {
				done <- true
			}
		})

		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_OUTPUT
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = c.Queue
		qObject, err := tbs.QMgr.Open(mqod, openOptions)
		if err != nil {
			log.Fatalf("error opening queue object %s, %s", c.Queue, err.Error())
		}

		putmqmd := ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
		buffer := []byte(msg)

		log.Printf("prepping queue %s with %d messages...", c.Queue, iterations)
		for i := 0; i < iterations; i++ {
			err = qObject.Put(putmqmd, pmo, buffer)
			if err != nil {
				log.Fatalf("error putting messages on queue")
			}
		}
		qObject.Close(0)
	}

	// Queues are ready, now start the bridge
	tbs.StartBridge(connect, false)

	<-done
	end := time.Now()

	stats := tbs.Bridge.SafeStats()
	statsJSON, _ := json.MarshalIndent(stats, "", "    ")

	// Close the test environ so we clean up the log
	tbs.Close()

	diff := end.Sub(start)
	rate := float64(maxCount) / float64(diff.Seconds())
	log.Printf("Bridge Stats:\n\n%s\n", statsJSON)
	log.Printf("Read %d messages from %d MQ queues via a bridge to NATS in %s, or %.2f msgs/sec", maxCount, len(connect), diff, rate)
}
