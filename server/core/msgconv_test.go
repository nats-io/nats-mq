package core

import (
	"bytes"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExcludeHeadersIgnoresMQMD(t *testing.T) {
	bridge := &BridgeServer{}
	msg := "hello world"
	msgBytes := []byte(msg)

	result, _, err := bridge.MQToNATSMessage(nil, EmptyHandle, msgBytes, len(msgBytes), nil)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	mqmd, _, result, err := bridge.NATSToMQMessage(msgBytes, "", nil)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	// mqmd should be default
	expected := ibmmq.NewMQMD()
	require.Equal(t, expected.Expiry, mqmd.Expiry)
	require.Equal(t, expected.Version, mqmd.Version)
	require.Equal(t, expected.OriginalLength, mqmd.OriginalLength)
	require.Equal(t, expected.Format, mqmd.Format)
}

func TestMQMDToNATSTranslation(t *testing.T) {
	bridge := &BridgeServer{}
	mqServer, qMgr, err := StartMQTestServer(5*time.Second, false)
	require.NoError(t, err)
	defer qMgr.Disc()
	defer mqServer.Close()

	msg := "hello world"
	msgBytes := []byte(msg)

	// Values aren't valid, but are testable
	expected := ibmmq.NewMQMD()
	expected.Version = 1
	expected.Report = 2
	expected.MsgType = 3
	expected.Expiry = 4
	expected.Feedback = 5
	expected.Encoding = 6
	expected.CodedCharSetId = 7
	expected.Format = "8"
	expected.Priority = 9
	expected.Persistence = ibmmq.MQPER_PERSISTENCE_AS_Q_DEF
	expected.MsgId = copyByteArray(msgBytes)
	expected.CorrelId = copyByteArray(msgBytes)
	expected.BackoutCount = 11
	expected.ReplyToQ = "12"
	expected.ReplyToQMgr = "13"
	expected.UserIdentifier = "14"
	expected.AccountingToken = copyByteArray(msgBytes)
	expected.ApplIdentityData = "15"
	expected.PutApplType = 16
	expected.PutApplName = "17"
	expected.PutDate = "18"
	expected.PutTime = "19"
	expected.ApplOriginData = "20"
	expected.GroupId = copyByteArray(msgBytes)
	expected.MsgSeqNumber = 21
	expected.Offset = 22
	expected.MsgFlags = 23
	expected.OriginalLength = 24

	cmho := ibmmq.NewMQCMHO()
	handleIn, err := qMgr.CrtMH(cmho)
	require.NoError(t, err)

	smpo := ibmmq.NewMQSMPO()
	pd := ibmmq.NewMQPD()
	err = handleIn.SetMP(smpo, "one", pd, "alpha")
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "two", pd, int(356))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "two8", pd, int8(17))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "two16", pd, int16(129))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "two32", pd, int32(357172))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "two64", pd, int64(11123123123))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "three32", pd, float32(3.0))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "three64", pd, float64(322222.0))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "four", pd, []byte("alpha"))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "five", pd, true)
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "six", pd, nil)
	require.NoError(t, err)

	encoded, _, err := bridge.MQToNATSMessage(expected, handleIn, msgBytes, len(msgBytes), qMgr)
	require.NoError(t, err)
	require.NotEqual(t, msg, string(encoded))

	mqmd, handleOut, result, err := bridge.NATSToMQMessage(encoded, "", qMgr)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	impo := ibmmq.NewMQIMPO()
	pd = ibmmq.NewMQPD()
	impo.Options = ibmmq.MQIMPO_CONVERT_VALUE
	_, value, err := handleOut.InqMP(impo, pd, "one")
	require.NoError(t, err)
	require.Equal(t, "alpha", value.(string))

	_, value, err = handleOut.InqMP(impo, pd, "two")
	require.NoError(t, err)
	require.Equal(t, int64(356), value.(int64))

	_, value, err = handleOut.InqMP(impo, pd, "two8")
	require.NoError(t, err)
	require.Equal(t, int8(17), value.(int8))

	_, value, err = handleOut.InqMP(impo, pd, "two16")
	require.NoError(t, err)
	require.Equal(t, int16(129), value.(int16))

	_, value, err = handleOut.InqMP(impo, pd, "two32")
	require.NoError(t, err)
	require.Equal(t, int32(357172), value.(int32))

	_, value, err = handleOut.InqMP(impo, pd, "two64")
	require.NoError(t, err)
	require.Equal(t, int64(11123123123), value.(int64))

	_, value, err = handleOut.InqMP(impo, pd, "three32")
	require.NoError(t, err)
	require.Equal(t, float32(3.0), value.(float32))

	_, value, err = handleOut.InqMP(impo, pd, "three64")
	require.NoError(t, err)
	require.Equal(t, float64(322222.0), value.(float64))

	_, value, err = handleOut.InqMP(impo, pd, "four")
	require.NoError(t, err)
	require.ElementsMatch(t, []byte("alpha"), value.([]byte))

	_, value, err = handleOut.InqMP(impo, pd, "five")
	require.NoError(t, err)
	require.Equal(t, true, value.(bool))

	_, value, err = handleOut.InqMP(impo, pd, "six")
	require.NoError(t, err)
	require.Nil(t, value)

	require.Equal(t, expected.OriginalLength, mqmd.OriginalLength)

	/* Some fields aren't copied, we will test some of these on 1 way
	require.Equal(t, expected.Version, mqmd.Version)
	require.Equal(t, expected.MsgType, mqmd.MsgType)
	require.Equal(t, expected.Expiry, mqmd.Expiry)
	require.Equal(t, expected.BackoutCount, mqmd.BackoutCount)
	require.Equal(t, expected.PutDate, mqmd.PutDate)
	require.Equal(t, expected.PutTime, mqmd.PutTime)
	*/
	require.Equal(t, expected.Persistence, mqmd.Persistence) // only works with the default
	require.Equal(t, expected.Report, mqmd.Report)
	require.Equal(t, expected.Feedback, mqmd.Feedback)
	require.Equal(t, expected.Encoding, mqmd.Encoding)
	require.Equal(t, expected.CodedCharSetId, mqmd.CodedCharSetId)
	require.Equal(t, expected.Format, mqmd.Format)
	require.Equal(t, expected.Priority, mqmd.Priority)
	require.Equal(t, expected.ReplyToQ, mqmd.ReplyToQ)
	require.Equal(t, expected.ReplyToQMgr, mqmd.ReplyToQMgr)
	require.Equal(t, expected.UserIdentifier, mqmd.UserIdentifier)
	require.Equal(t, expected.ApplIdentityData, mqmd.ApplIdentityData)
	require.Equal(t, expected.PutApplType, mqmd.PutApplType)
	require.Equal(t, expected.PutApplName, mqmd.PutApplName)
	require.Equal(t, expected.ApplOriginData, mqmd.ApplOriginData)
	require.Equal(t, expected.MsgSeqNumber, mqmd.MsgSeqNumber)
	require.Equal(t, expected.Offset, mqmd.Offset)
	require.Equal(t, expected.MsgFlags, mqmd.MsgFlags)
	require.Equal(t, expected.OriginalLength, mqmd.OriginalLength)

	require.ElementsMatch(t, expected.MsgId, bytes.Trim(mqmd.MsgId, "\x00"))
	require.ElementsMatch(t, expected.CorrelId, bytes.Trim(mqmd.CorrelId, "\x00"))
	require.ElementsMatch(t, expected.AccountingToken, bytes.Trim(mqmd.AccountingToken, "\x00"))
	require.ElementsMatch(t, expected.GroupId, bytes.Trim(mqmd.GroupId, "\x00"))
}

func TestNATSToMQMDTranslation(t *testing.T) {
	bridge := &BridgeServer{}
	mqServer, qMgr, err := StartMQTestServer(5*time.Second, false)
	require.NoError(t, err)
	defer qMgr.Disc()
	defer mqServer.Close()

	msg := "hello world"
	msgBytes := []byte(msg)

	// Values aren't valid, but are testable
	expected := message.NewBridgeMessage(msgBytes)
	expected.Header.Version = 1
	expected.Header.Report = 2
	expected.Header.MsgType = 3
	expected.Header.Expiry = 4
	expected.Header.Feedback = 5
	expected.Header.Encoding = 6
	expected.Header.CodedCharSetID = 7
	expected.Header.Format = "8"
	expected.Header.Priority = 9
	expected.Header.Persistence = ibmmq.MQPER_PERSISTENCE_AS_Q_DEF
	expected.Header.MsgID = copyByteArray(msgBytes)
	expected.Header.CorrelID = copyByteArray(msgBytes)
	expected.Header.BackoutCount = 11
	expected.Header.ReplyToQ = "12"
	expected.Header.ReplyToQMgr = "13"
	expected.Header.UserIdentifier = "14"
	expected.Header.AccountingToken = copyByteArray(msgBytes)
	expected.Header.ApplIdentityData = "15"
	expected.Header.PutApplType = 16
	expected.Header.PutApplName = "17"
	expected.Header.PutDate = "18"
	expected.Header.PutTime = "19"
	expected.Header.ApplOriginData = "20"
	expected.Header.GroupID = copyByteArray(msgBytes)
	expected.Header.MsgSeqNumber = 21
	expected.Header.Offset = 22
	expected.Header.MsgFlags = 23
	expected.Header.OriginalLength = 24

	expected.SetProperty("one", "alpha")
	expected.SetProperty("two", int(356))
	expected.SetProperty("two8", int8(17))
	expected.SetProperty("two16", int16(129))
	expected.SetProperty("two32", int32(357172))
	expected.SetProperty("two64", int64(11123123123))
	expected.SetProperty("three32", float32(3.0))
	expected.SetProperty("three64", float64(322222.0))
	expected.SetProperty("four", []byte("alpha"))
	expected.SetProperty("five", true)
	expected.SetProperty("six", nil)

	encodedBytes, err := expected.Encode()
	require.NoError(t, err)
	mqmd, handleOut, result, err := bridge.NATSToMQMessage(encodedBytes, "", qMgr)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	decodedBytes, _, err := bridge.MQToNATSMessage(mqmd, handleOut, result, len(result), qMgr)
	require.NoError(t, err)

	decoded, err := message.DecodeBridgeMessage(decodedBytes)
	require.NoError(t, err)
	require.Equal(t, msg, string(decoded.Body))

	strVal, ok := decoded.GetStringProperty("one")
	require.True(t, ok)
	require.Equal(t, "alpha", strVal)

	intVal, ok := decoded.GetInt64Property("two")
	require.True(t, ok)
	require.Equal(t, int64(356), intVal)

	int8Val, ok := decoded.GetInt8Property("two8")
	require.True(t, ok)
	require.Equal(t, int8(17), int8Val)

	int16Val, ok := decoded.GetInt16Property("two16")
	require.True(t, ok)
	require.Equal(t, int16(129), int16Val)

	int32Val, ok := decoded.GetInt32Property("two32")
	require.True(t, ok)
	require.Equal(t, int32(357172), int32Val)

	int64Val, ok := decoded.GetInt64Property("two64")
	require.True(t, ok)
	require.Equal(t, int64(11123123123), int64Val)

	float32Val, ok := decoded.GetFloat32Property("three32")
	require.True(t, ok)
	require.Equal(t, float32(3.0), float32Val)

	float64Val, ok := decoded.GetFloat64Property("three64")
	require.True(t, ok)
	require.Equal(t, float64(322222.0), float64Val)

	bytesVal, ok := decoded.GetBytesProperty("four")
	require.True(t, ok)
	require.ElementsMatch(t, []byte("alpha"), bytesVal)

	boolVal, ok := decoded.GetBoolProperty("five")
	require.True(t, ok)
	require.Equal(t, true, boolVal)

	nilVal, ok := decoded.GetTypedProperty("six")
	require.True(t, ok)
	require.Nil(t, nilVal)

	/* Some fields aren't copied, we will test some of these on 1 way
	require.Equal(t, expected.Version, mqmd.Version)
	require.Equal(t, expected.MsgType, mqmd.MsgType)
	require.Equal(t, expected.Expiry, mqmd.Expiry)
	require.Equal(t, expected.BackoutCount, mqmd.BackoutCount)
	require.Equal(t, expected.PutDate, mqmd.PutDate)
	require.Equal(t, expected.PutTime, mqmd.PutTime)
	*/
	require.Equal(t, expected.Header.Persistence, decoded.Header.Persistence) // only works with the default
	require.Equal(t, expected.Header.Report, decoded.Header.Report)
	require.Equal(t, expected.Header.Feedback, decoded.Header.Feedback)
	require.Equal(t, expected.Header.Encoding, decoded.Header.Encoding)
	require.Equal(t, expected.Header.CodedCharSetID, decoded.Header.CodedCharSetID)
	require.Equal(t, expected.Header.Format, decoded.Header.Format)
	require.Equal(t, expected.Header.Priority, decoded.Header.Priority)
	require.Equal(t, expected.Header.ReplyToQ, decoded.Header.ReplyToQ)
	require.Equal(t, expected.Header.ReplyToQMgr, decoded.Header.ReplyToQMgr)
	require.Equal(t, expected.Header.UserIdentifier, decoded.Header.UserIdentifier)
	require.Equal(t, expected.Header.ApplIdentityData, decoded.Header.ApplIdentityData)
	require.Equal(t, expected.Header.PutApplType, decoded.Header.PutApplType)
	require.Equal(t, expected.Header.PutApplName, decoded.Header.PutApplName)
	require.Equal(t, expected.Header.ApplOriginData, decoded.Header.ApplOriginData)
	require.Equal(t, expected.Header.MsgSeqNumber, decoded.Header.MsgSeqNumber)
	require.Equal(t, expected.Header.Offset, decoded.Header.Offset)
	require.Equal(t, expected.Header.MsgFlags, decoded.Header.MsgFlags)
	require.Equal(t, expected.Header.OriginalLength, decoded.Header.OriginalLength)

	require.ElementsMatch(t, expected.Header.MsgID, decoded.Header.MsgID)
	require.ElementsMatch(t, expected.Header.CorrelID, decoded.Header.CorrelID)
	require.ElementsMatch(t, expected.Header.AccountingToken, decoded.Header.AccountingToken)
	require.ElementsMatch(t, expected.Header.GroupID, decoded.Header.GroupID)
}
