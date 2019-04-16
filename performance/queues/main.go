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
	"strings"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nats-mq/nats-mq/core"
)

var iterations int
var queues string
var connName string
var channelName string
var qmgrName string

func main() {
	flag.IntVar(&iterations, "i", 1000, "iterations, docker image defaults to 5000 in queue")
	flag.StringVar(&queues, "q", "DEV.QUEUE.1,DEV.QUEUE.2,DEV.QUEUE.3", "comma separated list of queues, should match the subjects based on your bridge configuration")
	flag.StringVar(&connName, "conn", "localhost(1414)", "the MQ server connection name - localhost(1414)")
	flag.StringVar(&channelName, "chan", "DEV.APP.SVRCONN", "the MQ server channel name - DEV.APP.SVRCONN")
	flag.StringVar(&qmgrName, "qmgr", "QM1", "the qMgr name - QM1")
	flag.Parse()

	msg := strings.Repeat("stannats", 128) // 1024 bytes

	config := conf.MQConfig{
		ConnectionName: connName,
		ChannelName:    channelName,
		QueueManager:   qmgrName,
	}

	queueList := strings.Split(queues, ",")

	ready := sync.WaitGroup{}
	done := sync.WaitGroup{}
	starter := sync.WaitGroup{}

	starter.Add(1)

	for _, queue := range queueList {
		ready.Add(1)
		done.Add(1)

		go func(queue string) {
			qMgrForPub, err := core.ConnectToQueueManager(config)

			if err != nil || qMgrForPub == nil {
				log.Fatalf("unable to connect to queue manager")
			}

			qMgrForSub, err := core.ConnectToQueueManager(config)

			if err != nil || qMgrForSub == nil {
				log.Fatalf("unable to connect to queue manager")
			}

			mqod := ibmmq.NewMQOD()
			openOptions := ibmmq.MQOO_OUTPUT
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = queue
			qObjectForPub, err := qMgrForPub.Open(mqod, openOptions)
			if err != nil {
				log.Fatalf("error opening queue object %s, %s", queue, err.Error())
			}

			mqod = ibmmq.NewMQOD()
			openOptions = ibmmq.MQOO_INPUT_SHARED
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = queue
			qObjectForSub, err := qMgrForSub.Open(mqod, openOptions)
			if err != nil {
				log.Fatalf("error opening queue object %s, %s", queue, err.Error())
			}

			count := 0
			mqmd := ibmmq.NewMQMD()
			gmo := ibmmq.NewMQGMO()
			cmho := ibmmq.NewMQCMHO()
			propsMsgHandle, err := qMgrForSub.CrtMH(cmho)

			if err != nil {
				log.Fatalf("error creating message handle %s, %s", queue, err.Error())
			}

			gmo.MsgHandle = propsMsgHandle
			gmo.Options = ibmmq.MQGMO_SYNCPOINT
			gmo.Options |= ibmmq.MQGMO_WAIT
			gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
			gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE

			cbd := ibmmq.NewMQCBD()
			cbd.CallbackFunction = func(qMgr *ibmmq.MQQueueManager, hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
				if mqErr != nil && mqErr.MQCC != ibmmq.MQCC_OK {
					if mqErr.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
						return
					}
					log.Fatalf("mq error %s", queue)
					return
				}

				// ignore event calls
				if cbc != nil && cbc.CallType == ibmmq.MQCBCT_EVENT_CALL {
					return
				}

				qMgr.Cmit()

				count++
				if count%1000 == 0 {
					log.Printf("%s: count = %d", queue, count)
				}
				if count == iterations {
					done.Done()
				}
			}

			err = qObjectForSub.CB(ibmmq.MQOP_REGISTER, cbd, mqmd, gmo)

			if err != nil {
				log.Fatalf("error creating callback %s, %s", queue, err.Error())
			}

			ctlo := ibmmq.NewMQCTLO()
			ctlo.Options = ibmmq.MQCTLO_FAIL_IF_QUIESCING
			err = qMgrForSub.Ctl(ibmmq.MQOP_START, ctlo)

			if err != nil {
				log.Fatalf("error starting callback %s, %s", queue, err.Error())
			}

			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
			buffer := []byte(msg)

			log.Printf("sender ready for queue %s", queue)
			ready.Done()
			starter.Wait()
			log.Printf("sending %d messages through %s mq series...", iterations, queue)
			for i := 0; i < iterations; i++ {
				err = qObjectForPub.Put(putmqmd, pmo, buffer)
				if err != nil {
					log.Fatalf("error putting messages on queue, %s", err.Error())
				}
			}
		}(queue)
	}

	ready.Wait()
	start := time.Now()
	starter.Done()
	done.Wait()
	end := time.Now()

	count := iterations * len(queueList)
	diff := end.Sub(start)
	rate := float64(count) / float64(diff.Seconds())
	log.Printf("Sent %d messages through %d queues MQ callbacks in %s, or %.2f msgs/sec", count, len(queueList), diff, rate)
}
