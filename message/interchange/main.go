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
	"github.com/nats-io/nats-mq/message"
	"io/ioutil"
	"log"
)

var outputFile string

func main() {
	flag.StringVar(&outputFile, "o", "", "output filepath")
	flag.Parse()

	msg := message.NewBridgeMessage([]byte("hello world"))
	msg.Header = message.BridgeHeader{
		Version: 1,
		Report:  2,
		MsgID:   []byte("cafebabe"),
	}

	expected := map[string]interface{}{
		"string":  "hello world",
		"int8":    int8(9),
		"int16":   int16(259),
		"int32":   int32(222222222),
		"int64":   int64(222222222222222222),
		"float32": float32(3.14),
		"float64": float64(6.4999),
		"bool":    true,
		"bytes":   []byte("one two three four"),
	}

	for k, v := range expected {
		err := msg.SetProperty(k, v)
		if err != nil {
			log.Fatalf("error - %s", err.Error())
		}
	}

	bytes, err := msg.Encode()
	if err != nil {
		log.Fatalf("error - %s", err.Error())
	}

	err = ioutil.WriteFile(outputFile, bytes, 0644)
	if err != nil {
		log.Fatalf("error - %s", err.Error())
	}

	log.Printf("wrote interchange file to %s", outputFile)
}
