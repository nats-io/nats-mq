package main

import (
	"flag"
	"log"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/server/conf"
	"github.com/nats-io/nats-mq/server/core"
)

var iterations int

func main() {
	flag.IntVar(&iterations, "i", 1000, "iterations, docker image defaults to 5000 in queue")
	flag.Parse()

	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}
	tbs, err := core.StartTestEnvironment(connect)
	if err != nil {
		log.Fatalf("error starting test environment, %s", err.Error())
	}

	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queue
	qObject, err := tbs.QMgr.Open(mqod, openOptions)
	if err != nil {
		log.Fatalf("error opening queue object %s, %s", queue, err.Error())
	}
	defer qObject.Close(0)

	done := make(chan bool)
	count := 0

	tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		count++
		if count%1000 == 0 {
			log.Printf("received count = %d", count)
		}
		if count == iterations {
			done <- true
		}
	})

	putmqmd := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	buffer := []byte(msg)

	log.Printf("sending %d messages through the MQ to bridge to NATS...", iterations)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		err = qObject.Put(putmqmd, pmo, buffer)
	}
	<-done
	end := time.Now()

	diff := end.Sub(start)
	rate := float64(iterations) / float64(diff.Seconds())
	log.Printf("Sent %d messages through an MQ queue to a NATS subscriber in %s, or %.2f msgs/sec", iterations, diff, rate)
}
