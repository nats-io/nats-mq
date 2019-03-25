package io.nats.mqbridge;

import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Header {
    @JsonProperty("version")
    private int version;

    @JsonProperty("report")
    private int report;

    @JsonProperty("type")
    private int msgType;

    @JsonProperty("exp")
    private int expiry;

    @JsonProperty("feed")
    private int feedback;

    @JsonProperty("enc")
    private int encoding;

    @JsonProperty("charset")
    private int codedCharSetID;

    @JsonProperty("priority")
    private int priority;

    @JsonProperty("persist")
    private int persistence;

    @JsonProperty("backout")
    private int backoutCount;

    @JsonProperty("appl_type")
    private int putApplType;

    @JsonProperty("seq")
    private int msgSeqNumber;

    @JsonProperty("offset")
    private int offset;

    @JsonProperty("flags")
    private int msgFlags;

    @JsonProperty("orig_length")
    private int originalLength;

    @JsonProperty("msg_id")
    private byte[] msgID;

    @JsonProperty("corr_id")
    private byte[] correlID;

    @JsonProperty("acct_token")
    private byte[] accountingToken;

    @JsonProperty("grp_id")
    private byte[] groupID;

    @JsonProperty("format")
    private String format;

    @JsonProperty("rep_q")
    private String replyToQ;

    @JsonProperty("rep_qmgr")
    private String replyToQMgr;

    @JsonProperty("user_id")
    private String userIdentifier;

    @JsonProperty("appl_id")
    private String applIdentityData;

    @JsonProperty("appl_name")
    private String putApplName;

    @JsonProperty("date")
    private String putDate;

    @JsonProperty("time")
    private String putTime;

    @JsonProperty("appl_orig")
    private String applOriginData;

    @JsonProperty("reply_to_channel")
    private String replyToChannel;

    public Header() {
    }

    public Header(int version, int report, int msgType, int expiry, int feedback, int encoding, int codedCharSetID, int priority, int persistence, int backoutCount, int putApplType, int msgSeqNumber, int offset, int msgFlags, int originalLength, byte[] msgID, byte[] correlID, byte[] accountingToken, byte[] groupID, String format, String replyToQ, String replyToQMgr, String userIdentifier, String applIdentityData, String putApplName, String putDate, String putTime, String applOriginData, String replyToChannel) {
        this.version = version;
        this.report = report;
        this.msgType = msgType;
        this.expiry = expiry;
        this.feedback = feedback;
        this.encoding = encoding;
        this.codedCharSetID = codedCharSetID;
        this.priority = priority;
        this.persistence = persistence;
        this.backoutCount = backoutCount;
        this.putApplType = putApplType;
        this.msgSeqNumber = msgSeqNumber;
        this.offset = offset;
        this.msgFlags = msgFlags;
        this.originalLength = originalLength;
        this.msgID = msgID;
        this.correlID = correlID;
        this.accountingToken = accountingToken;
        this.groupID = groupID;
        this.format = format;
        this.replyToQ = replyToQ;
        this.replyToQMgr = replyToQMgr;
        this.userIdentifier = userIdentifier;
        this.applIdentityData = applIdentityData;
        this.putApplName = putApplName;
        this.putDate = putDate;
        this.putTime = putTime;
        this.applOriginData = applOriginData;
        this.replyToChannel = replyToChannel;
    }

    public int getVersion() {
        return this.version;
    }

    public void setVersion(int version) {
        this.version = version;
    }

    public int getReport() {
        return this.report;
    }

    public void setReport(int report) {
        this.report = report;
    }

    public int getMsgType() {
        return this.msgType;
    }

    public void setMsgType(int msgType) {
        this.msgType = msgType;
    }

    public int getExpiry() {
        return this.expiry;
    }

    public void setExpiry(int expiry) {
        this.expiry = expiry;
    }

    public int getFeedback() {
        return this.feedback;
    }

    public void setFeedback(int feedback) {
        this.feedback = feedback;
    }

    public int getEncoding() {
        return this.encoding;
    }

    public void setEncoding(int encoding) {
        this.encoding = encoding;
    }

    public int getCodedCharSetID() {
        return this.codedCharSetID;
    }

    public void setCodedCharSetID(int codedCharSetID) {
        this.codedCharSetID = codedCharSetID;
    }

    public int getPriority() {
        return this.priority;
    }

    public void setPriority(int priority) {
        this.priority = priority;
    }

    public int getPersistence() {
        return this.persistence;
    }

    public void setPersistence(int persistence) {
        this.persistence = persistence;
    }

    public int getBackoutCount() {
        return this.backoutCount;
    }

    public void setBackoutCount(int backoutCount) {
        this.backoutCount = backoutCount;
    }

    public int getPutApplType() {
        return this.putApplType;
    }

    public void setPutApplType(int putApplType) {
        this.putApplType = putApplType;
    }

    public int getMsgSeqNumber() {
        return this.msgSeqNumber;
    }

    public void setMsgSeqNumber(int msgSeqNumber) {
        this.msgSeqNumber = msgSeqNumber;
    }

    public int getOffset() {
        return this.offset;
    }

    public void setOffset(int offset) {
        this.offset = offset;
    }

    public int getMsgFlags() {
        return this.msgFlags;
    }

    public void setMsgFlags(int msgFlags) {
        this.msgFlags = msgFlags;
    }

    public int getOriginalLength() {
        return this.originalLength;
    }

    public void setOriginalLength(int originalLength) {
        this.originalLength = originalLength;
    }

    public byte[] getMsgID() {
        return this.msgID;
    }

    public void setMsgID(byte[] msgID) {
        this.msgID = msgID;
    }

    public byte[] getCorrelID() {
        return this.correlID;
    }

    public void setCorrelID(byte[] correlID) {
        this.correlID = correlID;
    }

    public byte[] getAccountingToken() {
        return this.accountingToken;
    }

    public void setAccountingToken(byte[] accountingToken) {
        this.accountingToken = accountingToken;
    }

    public byte[] getGroupID() {
        return this.groupID;
    }

    public void setGroupID(byte[] groupID) {
        this.groupID = groupID;
    }

    public String getFormat() {
        return this.format;
    }

    public void setFormat(String format) {
        this.format = format;
    }

    public String getReplyToQ() {
        return this.replyToQ;
    }

    public void setReplyToQ(String replyToQ) {
        this.replyToQ = replyToQ;
    }

    public String getReplyToQMgr() {
        return this.replyToQMgr;
    }

    public void setReplyToQMgr(String replyToQMgr) {
        this.replyToQMgr = replyToQMgr;
    }

    public String getUserIdentifier() {
        return this.userIdentifier;
    }

    public void setUserIdentifier(String userIdentifier) {
        this.userIdentifier = userIdentifier;
    }

    public String getApplIdentityData() {
        return this.applIdentityData;
    }

    public void setApplIdentityData(String applIdentityData) {
        this.applIdentityData = applIdentityData;
    }

    public String getPutApplName() {
        return this.putApplName;
    }

    public void setPutApplName(String putApplName) {
        this.putApplName = putApplName;
    }

    public String getPutDate() {
        return this.putDate;
    }

    public void setPutDate(String putDate) {
        this.putDate = putDate;
    }

    public String getPutTime() {
        return this.putTime;
    }

    public void setPutTime(String putTime) {
        this.putTime = putTime;
    }

    public String getApplOriginData() {
        return this.applOriginData;
    }

    public void setApplOriginData(String applOriginData) {
        this.applOriginData = applOriginData;
    }

    public String getReplyToChannel() {
        return this.replyToChannel;
    }

    public void setReplyToChannel(String replyToChannel) {
        this.replyToChannel = replyToChannel;
    }

    public Header version(int version) {
        this.version = version;
        return this;
    }

    public Header report(int report) {
        this.report = report;
        return this;
    }

    public Header msgType(int msgType) {
        this.msgType = msgType;
        return this;
    }

    public Header expiry(int expiry) {
        this.expiry = expiry;
        return this;
    }

    public Header feedback(int feedback) {
        this.feedback = feedback;
        return this;
    }

    public Header encoding(int encoding) {
        this.encoding = encoding;
        return this;
    }

    public Header codedCharSetID(int codedCharSetID) {
        this.codedCharSetID = codedCharSetID;
        return this;
    }

    public Header priority(int priority) {
        this.priority = priority;
        return this;
    }

    public Header persistence(int persistence) {
        this.persistence = persistence;
        return this;
    }

    public Header backoutCount(int backoutCount) {
        this.backoutCount = backoutCount;
        return this;
    }

    public Header putApplType(int putApplType) {
        this.putApplType = putApplType;
        return this;
    }

    public Header msgSeqNumber(int msgSeqNumber) {
        this.msgSeqNumber = msgSeqNumber;
        return this;
    }

    public Header offset(int offset) {
        this.offset = offset;
        return this;
    }

    public Header msgFlags(int msgFlags) {
        this.msgFlags = msgFlags;
        return this;
    }

    public Header originalLength(int originalLength) {
        this.originalLength = originalLength;
        return this;
    }

    public Header msgID(byte[] msgID) {
        this.msgID = msgID;
        return this;
    }

    public Header correlID(byte[] correlID) {
        this.correlID = correlID;
        return this;
    }

    public Header accountingToken(byte[] accountingToken) {
        this.accountingToken = accountingToken;
        return this;
    }

    public Header groupID(byte[] groupID) {
        this.groupID = groupID;
        return this;
    }

    public Header format(String format) {
        this.format = format;
        return this;
    }

    public Header replyToQ(String replyToQ) {
        this.replyToQ = replyToQ;
        return this;
    }

    public Header replyToQMgr(String replyToQMgr) {
        this.replyToQMgr = replyToQMgr;
        return this;
    }

    public Header userIdentifier(String userIdentifier) {
        this.userIdentifier = userIdentifier;
        return this;
    }

    public Header applIdentityData(String applIdentityData) {
        this.applIdentityData = applIdentityData;
        return this;
    }

    public Header putApplName(String putApplName) {
        this.putApplName = putApplName;
        return this;
    }

    public Header putDate(String putDate) {
        this.putDate = putDate;
        return this;
    }

    public Header putTime(String putTime) {
        this.putTime = putTime;
        return this;
    }

    public Header applOriginData(String applOriginData) {
        this.applOriginData = applOriginData;
        return this;
    }

    public Header replyToChannel(String replyToChannel) {
        this.replyToChannel = replyToChannel;
        return this;
    }

    @Override
    public boolean equals(Object o) {
        if (o == this)
            return true;
        if (!(o instanceof Header)) {
            return false;
        }
        Header header = (Header) o;
        return version == header.version && report == header.report && msgType == header.msgType && expiry == header.expiry && feedback == header.feedback && encoding == header.encoding && codedCharSetID == header.codedCharSetID && priority == header.priority && persistence == header.persistence && backoutCount == header.backoutCount && putApplType == header.putApplType && msgSeqNumber == header.msgSeqNumber && offset == header.offset && msgFlags == header.msgFlags && originalLength == header.originalLength && Objects.equals(msgID, header.msgID) && Objects.equals(correlID, header.correlID) && Objects.equals(accountingToken, header.accountingToken) && Objects.equals(groupID, header.groupID) && Objects.equals(format, header.format) && Objects.equals(replyToQ, header.replyToQ) && Objects.equals(replyToQMgr, header.replyToQMgr) && Objects.equals(userIdentifier, header.userIdentifier) && Objects.equals(applIdentityData, header.applIdentityData) && Objects.equals(putApplName, header.putApplName) && Objects.equals(putDate, header.putDate) && Objects.equals(putTime, header.putTime) && Objects.equals(applOriginData, header.applOriginData) && Objects.equals(replyToChannel, header.replyToChannel);
    }

    @Override
    public int hashCode() {
        return Objects.hash(version, report, msgType, expiry, feedback, encoding, codedCharSetID, priority, persistence, backoutCount, putApplType, msgSeqNumber, offset, msgFlags, originalLength, msgID, correlID, accountingToken, groupID, format, replyToQ, replyToQMgr, userIdentifier, applIdentityData, putApplName, putDate, putTime, applOriginData, replyToChannel);
    }

    @Override
    public String toString() {
        return "{" +
            " version='" + getVersion() + "'" +
            ", report='" + getReport() + "'" +
            ", msgType='" + getMsgType() + "'" +
            ", expiry='" + getExpiry() + "'" +
            ", feedback='" + getFeedback() + "'" +
            ", encoding='" + getEncoding() + "'" +
            ", codedCharSetID='" + getCodedCharSetID() + "'" +
            ", priority='" + getPriority() + "'" +
            ", persistence='" + getPersistence() + "'" +
            ", backoutCount='" + getBackoutCount() + "'" +
            ", putApplType='" + getPutApplType() + "'" +
            ", msgSeqNumber='" + getMsgSeqNumber() + "'" +
            ", offset='" + getOffset() + "'" +
            ", msgFlags='" + getMsgFlags() + "'" +
            ", originalLength='" + getOriginalLength() + "'" +
            ", msgID='" + getMsgID() + "'" +
            ", correlID='" + getCorrelID() + "'" +
            ", accountingToken='" + getAccountingToken() + "'" +
            ", groupID='" + getGroupID() + "'" +
            ", format='" + getFormat() + "'" +
            ", replyToQ='" + getReplyToQ() + "'" +
            ", replyToQMgr='" + getReplyToQMgr() + "'" +
            ", userIdentifier='" + getUserIdentifier() + "'" +
            ", applIdentityData='" + getApplIdentityData() + "'" +
            ", putApplName='" + getPutApplName() + "'" +
            ", putDate='" + getPutDate() + "'" +
            ", putTime='" + getPutTime() + "'" +
            ", applOriginData='" + getApplOriginData() + "'" +
            ", replyToChannel='" + getReplyToChannel() + "'" +
            "}";
    }
}