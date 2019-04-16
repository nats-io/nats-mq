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
 */

package core

import (
	"bytes"
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
)

// EmptyHandle is used when there is no message handle to pass in
var EmptyHandle ibmmq.MQMessageHandle = ibmmq.MQMessageHandle{}

// Copies the array, empty or not
func copyByteArray(data []byte) []byte {
	newArray := make([]byte, len(data))
	copy(newArray, data)
	newArray = bytes.Trim(newArray, "\x00")
	return newArray
}

// Copies the array if it isn't empty, otherwise returns the default
func copyByteArrayIfNotEmpty(data []byte, def []byte, size int32) []byte {
	if len(data) == 0 {
		return def
	}
	newArray := bytes.Repeat([]byte{0}, int(size))
	copy(newArray, data)
	return newArray
}

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
	mqmd.MsgId = copyByteArrayIfNotEmpty(header.MsgID, mqmd.MsgId, ibmmq.MQ_MSG_ID_LENGTH)
	mqmd.CorrelId = copyByteArrayIfNotEmpty(header.CorrelID, mqmd.CorrelId, ibmmq.MQ_CORREL_ID_LENGTH)
	mqmd.ReplyToQ = header.ReplyToQ
	mqmd.ReplyToQMgr = header.ReplyToQMgr
	mqmd.UserIdentifier = header.UserIdentifier
	mqmd.AccountingToken = copyByteArrayIfNotEmpty(header.AccountingToken, mqmd.AccountingToken, ibmmq.MQ_ACCOUNTING_TOKEN_LENGTH)
	mqmd.ApplIdentityData = header.ApplIdentityData
	mqmd.PutApplType = header.PutApplType
	mqmd.PutApplName = header.PutApplName
	mqmd.ApplOriginData = header.ApplOriginData
	mqmd.GroupId = copyByteArrayIfNotEmpty(header.GroupID, mqmd.GroupId, ibmmq.MQ_GROUP_ID_LENGTH)
	mqmd.MsgSeqNumber = header.MsgSeqNumber
	mqmd.Offset = header.Offset
	mqmd.MsgFlags = header.MsgFlags
	mqmd.OriginalLength = header.OriginalLength

	return mqmd
}

func (bridge *BridgeServer) copyMessageProperties(handle ibmmq.MQMessageHandle, msg *message.BridgeMessage) error {
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

func (bridge *BridgeServer) mapPropertiesToHandle(msg *message.BridgeMessage, qmgr *ibmmq.MQQueueManager) (ibmmq.MQMessageHandle, error) {
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
			return handle, fmt.Errorf("broken message property %s", name)
		}

		err = handle.SetMP(smpo, name, pd, value)
		if err != nil {
			return handle, err
		}
	}

	return handle, nil
}

//MQToNATSMessage convert an incoming MQ message to a set of NATS bytes and a reply subject
// if the qmgr is nil, the return value is just the message body
// if the qmgr is not nil the message is encoded as a BridgeMessage
// The data array is always just bytes from MQ, and is not an encoded BridgeMessage
// Header fields that are byte arrays are trimmed, "\x00" removed, on conversion to BridgeMessage.Header
func (bridge *BridgeServer) MQToNATSMessage(mqmd *ibmmq.MQMD, handle ibmmq.MQMessageHandle, data []byte, length int, qmgr *ibmmq.MQQueueManager) ([]byte, string, error) {
	replySubject := ""
	replyChannel := ""
	replyQ := ""
	replyQMgr := ""

	if mqmd != nil {
		replyQ = mqmd.ReplyToQ
		replyQMgr = mqmd.ReplyToQMgr
	}

	if replyQ != "" && replyQMgr != "" {
		connectTo, ok := bridge.replyToInfo["Q:"+replyQ+"@"+replyQMgr]

		if ok {
			if connectTo.Subject != "" {
				replySubject = connectTo.Subject
			} else {
				replyChannel = connectTo.Channel
			}
		}
	}

	if qmgr == nil {
		return data[:length], replySubject, nil
	}

	mqMsg := message.NewBridgeMessage(data[:length])

	mqMsg.Header = mapMQMDToHeader(mqmd)
	mqMsg.Header.ReplyToChannel = replyChannel

	err := bridge.copyMessageProperties(handle, mqMsg)

	if err != nil {
		return nil, "", err
	}

	encoded, err := mqMsg.Encode()

	if err != nil {
		return nil, "", err
	}

	return encoded, replySubject, nil
}

// NATSToMQMessage decode an incoming nats message to an MQ message
// if the qmgr is nil, data is considered to just be a message body
// if the qmgr is not nil the message is treated as an encoded BridgeMessage
// The returned byte array just bytes from MQ, and is not an encoded BridgeMessage
// Header fields that are byte arrays are padded, "\x00" added, on conversion from BridgeMessage.Header
func (bridge *BridgeServer) NATSToMQMessage(data []byte, replyTo string, qmgr *ibmmq.MQQueueManager) (*ibmmq.MQMD, ibmmq.MQMessageHandle, []byte, error) {
	replyQ := ""
	replyQMgr := ""

	if replyTo != "" {
		connectTo, ok := bridge.replyToInfo["S:"+replyTo]

		if !ok {
			connectTo, ok = bridge.replyToInfo["C:"+replyTo]
		}

		if ok && connectTo.Queue != "" {
			replyQ = connectTo.Queue
			replyQMgr = connectTo.MQ.QueueManager
		}
	}

	if qmgr == nil {
		mqmd := ibmmq.NewMQMD()

		if replyQ != "" {
			mqmd.ReplyToQ = replyQ
			mqmd.ReplyToQMgr = replyQMgr
		}

		return mqmd, EmptyHandle, data, nil
	}

	// Can't have nil data for encoded message, could have for empty plain message
	if data == nil {
		return nil, EmptyHandle, nil, fmt.Errorf("tried to convert empty message to BridgeMessage")
	}

	mqMsg, err := message.DecodeBridgeMessage(data)

	if err != nil {
		return nil, EmptyHandle, nil, err
	}

	if mqMsg.Header.ReplyToChannel != "" {
		connectTo, ok := bridge.replyToInfo["C:"+mqMsg.Header.ReplyToChannel]
		if ok && connectTo.Queue != "" {
			replyQ = connectTo.Queue
			replyQMgr = connectTo.MQ.QueueManager
		}
	}

	handle, err := bridge.mapPropertiesToHandle(mqMsg, qmgr)

	if err != nil {
		return nil, EmptyHandle, nil, err
	}

	mqmd := mapHeaderToMQMD(&mqMsg.Header)

	if replyQ != "" {
		mqmd.ReplyToQ = replyQ
		mqmd.ReplyToQMgr = replyQMgr
	}

	return mqmd, handle, mqMsg.Body, nil
}
