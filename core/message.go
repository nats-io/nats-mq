package core

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/ugorji/go/codec"
)

var mh codec.MsgpackHandle

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

// NewMQBridgeHeader creates a new bridge header from an MQMD, copying all the contents
func NewMQBridgeHeader(mqmd *ibmmq.MQMD) MQBridgeHeader {
	return MQBridgeHeader{
		Version:          mqmd.Version,
		Report:           mqmd.Report,
		MsgType:          mqmd.MsgType,
		Expiry:           mqmd.Expiry,
		Feedback:         mqmd.Feedback,
		Encoding:         mqmd.Encoding,
		CodedCharSetID:   mqmd.CodedCharSetId,
		Format:           mqmd.Format,
		Priority:         mqmd.Priority,
		Persistence:      mqmd.Persistence,
		MsgID:            copyByteArray(mqmd.MsgId),
		CorrelID:         copyByteArray(mqmd.CorrelId),
		BackoutCount:     mqmd.BackoutCount,
		ReplyToQ:         mqmd.ReplyToQ,
		ReplyToQMgr:      mqmd.ReplyToQMgr,
		UserIdentifier:   mqmd.UserIdentifier,
		AccountingToken:  copyByteArray(mqmd.AccountingToken),
		ApplIdentityData: mqmd.ApplIdentityData,
		PutApplType:      mqmd.PutApplType,
		PutApplName:      mqmd.PutApplName,
		PutDate:          mqmd.PutDate,
		PutTime:          mqmd.PutTime,
		ApplOriginData:   mqmd.ApplOriginData,
		GroupID:          copyByteArray(mqmd.GroupId),
		MsgSeqNumber:     mqmd.MsgSeqNumber,
		Offset:           mqmd.Offset,
		MsgFlags:         mqmd.MsgFlags,
		OriginalLength:   mqmd.OriginalLength,
	}
}

// Copy all of the fields, but many will be ignored on Put
func (header *MQBridgeHeader) toMQMD() *ibmmq.MQMD {
	mqmd := ibmmq.NewMQMD()

	/* some fields shouldn't be copied, they aren't user editable
	mqmd.Version = header.Version
	mqmd.MsgType = header.MsgType
	mqmd.Expiry = header.Expiry
	mqmd.BackoutCount = header.BackoutCount
	mqmd.Persistence = header.Persistence
	mqmd.PutDate = header.PutDate
	mqmd.PutTime = header.PutTime
	*/
	mqmd.Report = header.Report
	mqmd.Feedback = header.Feedback
	mqmd.Encoding = header.Encoding
	mqmd.CodedCharSetId = header.CodedCharSetID
	mqmd.Format = header.Format
	mqmd.Priority = header.Priority
	mqmd.MsgId = copyByteArrayIfNotEmpty(header.MsgID, mqmd.MsgId)
	mqmd.CorrelId = copyByteArrayIfNotEmpty(header.CorrelID, mqmd.CorrelId)
	mqmd.ReplyToQ = header.ReplyToQ
	mqmd.ReplyToQMgr = header.ReplyToQMgr
	mqmd.UserIdentifier = header.UserIdentifier
	mqmd.AccountingToken = copyByteArrayIfNotEmpty(header.AccountingToken, mqmd.AccountingToken)
	mqmd.ApplIdentityData = header.ApplIdentityData
	mqmd.PutApplType = header.PutApplType
	mqmd.PutApplName = header.PutApplName
	mqmd.ApplOriginData = header.ApplOriginData
	mqmd.GroupId = copyByteArrayIfNotEmpty(header.GroupID, mqmd.GroupId)
	mqmd.MsgSeqNumber = header.MsgSeqNumber
	mqmd.Offset = header.Offset
	mqmd.MsgFlags = header.MsgFlags
	mqmd.OriginalLength = header.OriginalLength

	return mqmd
}

// MQBridgeMessage is the NATS-side wrapper for the mq message
type MQBridgeMessage struct {
	Header MQBridgeHeader `codec:"header,omitempty"`
	Body   []byte         `codec:"body,omitempty"`
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
func NewBridgeMessage(data []byte) *MQBridgeMessage {
	return &MQBridgeMessage{
		Body: data,
	}
}

// NewBridgeMessageFromBytes decodes the bytes (if exclude headers is false) and returns the decoded version
// if exclude headers is true, the returned message will have an empty header
func NewBridgeMessageFromBytes(data []byte, excludeHeaders bool) (*MQBridgeMessage, error) {
	mqMsg := &MQBridgeMessage{}

	if excludeHeaders {
		mqMsg.Body = data
		return mqMsg, nil
	}

	dec := codec.NewDecoderBytes(data, &mh)
	err := dec.Decode(mqMsg)

	if err != nil {
		return nil, err
	}

	return mqMsg, nil
}

//MQToNATSMessage convert an incoming MQ message to a set of NATS bytes
func MQToNATSMessage(mqmd *ibmmq.MQMD, data []byte, len int, excludeHeaders bool) ([]byte, error) {

	if excludeHeaders {
		return data[:len], nil
	}

	mqMsg := &MQBridgeMessage{
		Header: NewMQBridgeHeader(mqmd),
		Body:   data[:len],
	}

	var raw [1024]byte
	b := raw[:0]
	enc := codec.NewEncoderBytes(&b, &mh)
	err := enc.Encode(mqMsg)

	if err != nil {
		return nil, err
	}

	return b, nil
}

// NATSToMQMessage decode an incoming nats message to an MQ message
func NATSToMQMessage(data []byte, excludeHeaders bool) ([]byte, *ibmmq.MQMD, error) {

	if excludeHeaders {
		return data, ibmmq.NewMQMD(), nil
	}

	mqMsg := &MQBridgeMessage{}
	dec := codec.NewDecoderBytes(data, &mh)
	err := dec.Decode(mqMsg)

	if err != nil {
		return nil, nil, err
	}

	return mqMsg.Body, mqMsg.Header.toMQMD(), nil
}

// Copies the array, empty or not
func copyByteArray(bytes []byte) []byte {
	newArray := make([]byte, len(bytes))
	copy(newArray, bytes)
	return newArray
}

// Copies the array if it isn't empty, otherwise returns the default
func copyByteArrayIfNotEmpty(bytes []byte, def []byte) []byte {
	if bytes == nil || len(bytes) == 0 {
		return def
	}
	newArray := make([]byte, len(bytes))
	copy(newArray, bytes)
	return newArray
}
