package core

import (
	"fmt"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
)

// EmptyHandle is used when there is no message handle to pass in
var EmptyHandle ibmmq.MQMessageHandle = ibmmq.MQMessageHandle{}

// mapMQMDToHeader creates a new bridge header from an MQMD, copying all the contents
func mapMQMDToHeader(mqmd *ibmmq.MQMD) message.BridgeHeader {
	return message.BridgeHeader{
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

// mapHeaderToMQMD copies most of the fields, some will be ignored on Put, fields that cannot be set are skiped
func mapHeaderToMQMD(header *message.BridgeHeader) *ibmmq.MQMD {
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

func copyMessageProperties(handle ibmmq.MQMessageHandle, msg *message.BridgeMessage) error {
	if handle == EmptyHandle {
		return nil
	}

	impo := ibmmq.NewMQIMPO()
	pd := ibmmq.NewMQPD()

	impo.Options = ibmmq.MQIMPO_CONVERT_VALUE | ibmmq.MQIMPO_INQ_FIRST
	for propsToRead := true; propsToRead; {
		name, value, err := handle.InqMP(impo, pd, "%")
		impo.Options = ibmmq.MQIMPO_CONVERT_VALUE | ibmmq.MQIMPO_INQ_NEXT
		if err != nil {
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC != ibmmq.MQRC_PROPERTY_NOT_AVAILABLE {
				return err
			}
			propsToRead = false
		} else {
			err := msg.SetProperty(name, value) // will extract the type
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func mapPropertiesToHandle(msg *message.BridgeMessage, qmgr *ibmmq.MQQueueManager) (ibmmq.MQMessageHandle, error) {
	cmho := ibmmq.NewMQCMHO()
	handle, err := qmgr.CrtMH(cmho)
	if err != nil {
		return handle, err
	}

	smpo := ibmmq.NewMQSMPO()
	pd := ibmmq.NewMQPD()

	props := msg.Properties

	for name := range props {
		value, ok := msg.GetTypedProperty(name)

		if !ok {
			return handle, fmt.Errorf("encountered broken message property %s", name)
		}

		err = handle.SetMP(smpo, name, pd, value)
		if err != nil {
			return handle, err
		}
	}

	return handle, nil
}

//mqToNATSMessage convert an incoming MQ message to a set of NATS bytes
// if the qmgr is nil, the return value is just the message body
// if the qmgr is not nil the message is encoded as a BridgeMessage
func mqToNATSMessage(mqmd *ibmmq.MQMD, handle ibmmq.MQMessageHandle, data []byte, len int, qmgr *ibmmq.MQQueueManager) ([]byte, error) {
	if qmgr == nil {
		return data[:len], nil
	}

	mqMsg := message.NewBridgeMessage(data[:len])
	mqMsg.Header = mapMQMDToHeader(mqmd)

	err := copyMessageProperties(handle, mqMsg)

	if err != nil {
		return nil, err
	}

	return mqMsg.Encode()
}

// natsToMQMessage decode an incoming nats message to an MQ message
// if the qmgr is nil, data is considered to just be a message body
// if the qmgr is not nil the message is treated as an encoded BridgeMessage
func natsToMQMessage(data []byte, qmgr *ibmmq.MQQueueManager) (*ibmmq.MQMD, ibmmq.MQMessageHandle, []byte, error) {

	if qmgr == nil {
		return ibmmq.NewMQMD(), EmptyHandle, data, nil
	}

	mqMsg, err := message.DecodeBridgeMessage(data)

	if err != nil {
		return nil, EmptyHandle, nil, err
	}

	handle, err := mapPropertiesToHandle(mqMsg, qmgr)

	if err != nil {
		return nil, EmptyHandle, nil, err
	}

	return mapHeaderToMQMD(&mqMsg.Header), handle, mqMsg.Body, nil
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
