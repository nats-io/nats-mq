package main

import (
	"flag"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nats-mq/nats-mq/core"
)

var iterations int
var queues string
var subjects string
var connName string
var channelName string
var qmgrName string
var natsURL string

func main() {
	flag.IntVar(&iterations, "i", 1000, "iterations, docker image defaults to 5000 in queue")
	flag.StringVar(&queues, "q", "DEV.QUEUE.1,DEV.QUEUE.2,DEV.QUEUE.3", "comma separated list of queues, should match the subjects based on your bridge configuration")
	flag.StringVar(&subjects, "s", "one,two,three", "comma separated list of subjects, should match the queues based on your bridge configuration")
	flag.StringVar(&connName, "conn", "localhost(1414)", "the MQ server connection name - localhost(1414)")
	flag.StringVar(&channelName, "chan", "DEV.APP.SVRCONN", "the MQ server channel name - DEV.APP.SVRCONN")
	flag.StringVar(&qmgrName, "qmgr", "QM1", "the qMgr name - QM1")
	flag.StringVar(&natsURL, "nats", "localhost:4222", "nats url - localhost:4222")
	flag.Parse()

	msg := strings.Repeat("stannats", 128) // 1024 bytes

	config := conf.MQConfig{
		ConnectionName: connName,
		ChannelName:    channelName,
		QueueManager:   qmgrName,
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("unable to connect to nats")
	}

	queueList := strings.Split(queues, ",")
	subjectList := strings.Split(subjects, ",")

	ready := sync.WaitGroup{}
	done := sync.WaitGroup{}
	starter := sync.WaitGroup{}

	starter.Add(1)

	for i, queue := range queueList {
		ready.Add(1)
		done.Add(1)
		subject := subjectList[i]

		go func(queue, subject string) {
			qMgr, err := core.ConnectToQueueManager(config)

			if err != nil || qMgr == nil {
				log.Fatalf("unable to connect to queue manager")
			}

			mqod := ibmmq.NewMQOD()
			openOptions := ibmmq.MQOO_OUTPUT
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = queue
			qObject, err := qMgr.Open(mqod, openOptions)
			if err != nil {
				log.Fatalf("error opening queue object %s, %s", queue, err.Error())
			}
			defer qObject.Close(0)

			count := 0

			nc.Subscribe(subject, func(msg *nats.Msg) {
				count++
				if count%1000 == 0 {
					log.Printf("%s: count = %d", subject, count)
				}
				if count == iterations {
					done.Done()
				}
			})

			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
			buffer := []byte(msg)

			log.Printf("sender ready for queue %s", queue)
			ready.Done()
			starter.Wait()
			log.Printf("sending %d messages through %s to bridge to NATS...", iterations, queue)
			for i := 0; i < iterations; i++ {
				err = qObject.Put(putmqmd, pmo, buffer)
				if err != nil {
					log.Fatalf("error putting messages on queue")
				}
			}
		}(queue, subject)
	}

	ready.Wait()
	start := time.Now()
	starter.Done()
	done.Wait()
	end := time.Now()

	count := iterations * len(queueList)
	diff := end.Sub(start)
	rate := float64(count) / float64(diff.Seconds())
	log.Printf("Sent %d messages through %d queues to a NATS subscribers in %s, or %.2f msgs/sec", count, len(queueList), diff, rate)
}
