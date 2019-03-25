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
