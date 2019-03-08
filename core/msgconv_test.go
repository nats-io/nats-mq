package core

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExcludeHeadersIgnoresMQMD(t *testing.T) {
	msg := "hello world"
	msgBytes := []byte(msg)

	result, err := mqToNATSMessage(nil, EmptyHandle, msgBytes, len(msgBytes), nil)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	mqmd, _, result, err := natsToMQMessage(msgBytes, nil)
	require.NoError(t, err)
	require.Equal(t, msg, string(result))

	// mqmd should be default
	expected := ibmmq.NewMQMD()
	require.Equal(t, expected.Expiry, mqmd.Expiry)
	require.Equal(t, expected.Version, mqmd.Version)
	require.Equal(t, expected.OriginalLength, mqmd.OriginalLength)
	require.Equal(t, expected.Format, mqmd.Format)
}

func TestMQMDTranslation(t *testing.T) {
	mqServer, qMgr, err := StartMQTestServer(5 * time.Second)
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
	expected.Persistence = 10
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
	err = handleIn.SetMP(smpo, "two", pd, int32(356))
	require.NoError(t, err)
	err = handleIn.SetMP(smpo, "three", pd, float32(3.0))
	require.NoError(t, err)

	encoded, err := mqToNATSMessage(expected, handleIn, msgBytes, len(msgBytes), qMgr)
	require.NoError(t, err)
	require.NotEqual(t, msg, string(encoded))

	mqmd, handleOut, result, err := natsToMQMessage(encoded, qMgr)
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

	_, value, err = handleOut.InqMP(impo, pd, "three")
	require.NoError(t, err)
	require.Equal(t, 3.0, value.(float64))

	require.Equal(t, expected.OriginalLength, mqmd.OriginalLength)

	/* Some fields aren't copied, we will test some of these on 1 way
	require.Equal(t, expected.Version, mqmd.Version)
	require.Equal(t, expected.MsgType, mqmd.MsgType)
	require.Equal(t, expected.Expiry, mqmd.Expiry)
	require.Equal(t, expected.BackoutCount, mqmd.BackoutCount)
	require.Equal(t, expected.Persistence, mqmd.Persistence)
	require.Equal(t, expected.PutDate, mqmd.PutDate)
	require.Equal(t, expected.PutTime, mqmd.PutTime)
	*/
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

	require.ElementsMatch(t, expected.MsgId, mqmd.MsgId)
	require.ElementsMatch(t, expected.CorrelId, mqmd.CorrelId)
	require.ElementsMatch(t, expected.AccountingToken, mqmd.AccountingToken)
	require.ElementsMatch(t, expected.GroupId, mqmd.GroupId)
}
