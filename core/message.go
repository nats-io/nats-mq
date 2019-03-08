package core

import (
	"github.com/ugorji/go/codec"
)

var mh codec.MsgpackHandle

func init() {
	mh.RawToString = true
	mh.WriteExt = true
	mh.SignedInteger = true
}

// MQBridgeHeader maps to an MQMD struct in the MQ messages
type MQBridgeHeader struct {
	Version          int32
	Report           int32
	MsgType          int32
	Expiry           int32
	Feedback         int32
	Encoding         int32
	CodedCharSetID   int32
	Format           string
	Priority         int32
	Persistence      int32
	MsgID            []byte
	CorrelID         []byte
	BackoutCount     int32
	ReplyToQ         string
	ReplyToQMgr      string
	UserIdentifier   string
	AccountingToken  []byte
	ApplIdentityData string
	PutApplType      int32
	PutApplName      string
	PutDate          string
	PutTime          string
	ApplOriginData   string
	GroupID          []byte
	MsgSeqNumber     int32
	Offset           int32
	MsgFlags         int32
	OriginalLength   int32
}

// MQBridgeMessage is the NATS-side wrapper for the mq message
type MQBridgeMessage struct {
	Header     MQBridgeHeader         `codec:"header,omitempty"`
	Properties map[string]interface{} `codec:"props,omitempty"`
	Body       []byte                 `codec:"body,omitempty"`
}

// Encode a bridge message to bytes
func (msg *MQBridgeMessage) Encode() ([]byte, error) {
	var raw [1024]byte
	b := raw[:0]
	enc := codec.NewEncoderBytes(&b, &mh)
	err := enc.Encode(msg)

	if err != nil {
		return nil, err
	}

	return b, nil
}

// NewBridgeMessage creates an empty message with the provided body, the header is empty
func NewBridgeMessage(body []byte) *MQBridgeMessage {
	return &MQBridgeMessage{
		Body: body,
	}
}

// DecodeBridgeMessage decodes the bytes and returns the decoded version
// use NewBridgeMessage to create a message with an empty header
func DecodeBridgeMessage(data []byte) (*MQBridgeMessage, error) {
	mqMsg := &MQBridgeMessage{}
	dec := codec.NewDecoderBytes(data, &mh)
	err := dec.Decode(mqMsg)

	if err != nil {
		return nil, err
	}

	return mqMsg, nil
}
