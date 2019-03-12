package main

import (
	"flag"
	"log"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/conf"
	"github.com/nats-io/nats-mq/core"
	"github.com/nats-io/nats-mq/examples"
)

var configFile string
var queue string
var subject string
var iterations int

func main() {
	flag.StringVar(&configFile, "c", "", "configuration filepath")
	flag.StringVar(&queue, "q", "", "queue name")
	flag.StringVar(&subject, "s", "", "subject")
	flag.IntVar(&iterations, "i", 100, "iterations")
	flag.Parse()

	if configFile == "" {
		log.Fatalf("no config file specified")
	}

	if queue == "" {
		log.Fatalf("no queue specified")
	}

	if subject == "" {
		log.Fatalf("no subject specified")
	}

	config := examples.ExampleConfiguration{}
	log.Printf("loading configuration from %q", configFile)

	if err := conf.LoadConfigFromFile(configFile, &config, false); err != nil {
		log.Fatalf("error loading config file %s", err.Error())
	}

	nc, err := core.ConnectToNATSWithConfig(config.NATS)
	if err != nil {
		log.Fatalf("error connecting to NATS %s", err.Error())
	}
	defer nc.Close()
	log.Println("connected to NATS")

	mq, err := core.ConnectToQueueManager(config.MQ)
	if err != nil {
		log.Fatalf("error connecting to MQ %s", err.Error())
	}
	defer mq.Disc()
	log.Println("connected to MQ")

	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queue
	qObject, err := mq.Open(mqod, openOptions)
	if err != nil {
		log.Fatalf("error opening queue object %s, %s", queue, err.Error())
	}
	defer qObject.Close(0)

	done := make(chan bool)
	count := 0

	nc.Subscribe(subject, func(msg *nats.Msg) {
		count++
		if count%1000 == 0 {
			log.Printf("count = %d", count)
		}
		if count == iterations {
			done <- true
		}
	})

	log.Printf("sending %d messages through the bridge...", iterations)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		putmqmd := ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
		putmqmd.Format = ibmmq.MQFMT_STRING
		msgData := "hello"
		buffer := []byte(msgData)
		err = qObject.Put(putmqmd, pmo, buffer)
	}
	<-done
	end := time.Now()

	diff := end.Sub(start)
	rate := float64(iterations) / float64(diff.Seconds())
	log.Printf("Sent %d messages through an MQ queue to a NATS subscriber in %s, or %.2f msgs/sec", iterations, diff, rate)
}
