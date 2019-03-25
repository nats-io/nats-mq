# NATS-MQ Bridge Message Format

The bridge provides two modes of message handling. In the [ExcludeHeaders](config.md#connectors) mode, a connector will take the raw NATS messages and put them into MQ messages as the body, or vice versa. No translation occurs and all MQ headers and properties are ignored. If ExcludeHeaders is false, the default, MQ messages are translated into a [msgpack](https://msgpack.org/index.html) format. NATS clients are required to use this format as well when in this mode.

The remainder of this document is focused on the message format when ExcludeHeaders is false, and encoding occurs.

* [Encoded Messages](#encode)
  * [Known Headers/Metadata](#headers)
  * [Message Properties](#props)
  * [The Message Body](#body)
* [Request-Reply](#reqrep)
* [Helpers](#helpers)
  * [Golang](#golang)

<a name="encode"></a>

## Encoded Messages

Encoded messages have three root level elements:

* Properties - `props` - mapped to/from MQ series message properties. These are typed, and have some type limitations.
* Header - `header` - a structure containing the MQ series message headers/metadata.
* Body - `body` - the byte array body of the MQ message.

It is worth thinking about the encoding process from two sides. When messages come out of MQ series, the bridge can read all of the properties and create a valid map of them. The bridge can also read all of the known headers and collect them. Of course, the message body can be read as well, although some size limits may be encountered on the NATS side. In other words, messages coming from MQ series should map well to the encoded format. Messages created in NATS and sent to the bridge, as msgpack encoded byte arrays may have some restrictions. For example, the `PutDate` header can't be set by a client so it is ignored when moving through the bridge into MQ series.

<a name="headers"></a>

### Known Headers/Metadata

The encoded header structure contains the following elements (in golang):

```golang
type BridgeHeader struct {
    Version          int32  `codec:"version,omitempty"`
    Report           int32  `codec:"report,omitempty"`
    MsgType          int32  `codec:"type,omitempty"`
    Expiry           int32  `codec:"exp,omitempty"`
    Feedback         int32  `codec:"feed,omitempty"`
    Encoding         int32  `codec:"enc,omitempty"`
    CodedCharSetID   int32  `codec:"charset,omitempty"`
    Format           string `codec:"format,omitempty"`
    Priority         int32  `codec:"priority,omitempty"`
    Persistence      int32  `codec:"persist,omitempty"`
    MsgID            []byte `codec:"msg_id,omitempty"`
    CorrelID         []byte `codec:"corr_id,omitempty"`
    BackoutCount     int32  `codec:"backout,omitempty"`
    ReplyToQ         string `codec:"rep_q,omitempty"`
    ReplyToQMgr      string `codec:"rep_qmgr,omitempty"`
    UserIdentifier   string `codec:"user_id,omitempty"`
    AccountingToken  []byte `codec:"acct_token,omitempty"`
    ApplIdentityData string `codec:"appl_id,omitempty"`
    PutApplType      int32  `codec:"appl_type,omitempty"`
    PutApplName      string `codec:"appl_name,omitempty"`
    PutDate          string `codec:"date,omitempty"`
    PutTime          string `codec:"time,omitempty"`
    ApplOriginData   string `codec:"appl_orig_data,omitempty"`
    GroupID          []byte `codec:"grp_id,omitempty"`
    MsgSeqNumber     int32  `codec:"seq,omitempty"`
    Offset           int32  `codec:"offset,omitempty"`
    MsgFlags         int32  `codec:"flags,omitempty"`
    OriginalLength   int32  `codec:"orig_length,omitempty"`
    ReplyToChannel   string `codec:"reply_to_channel,omitempty"`
}
```

These will be encoded into msgpack with their full field names and types.

<a name="props"></a>

### Message Properties

Because message properties are typed, we encode them into a special struct. The format in Go is:

```golang
type Property struct {
    Type  int         `codec:"type,omitempty"`
    Value interface{} `codec:"value,omitempty"`
}
```

Where the type is one of the following self-describing values:

```golang
PropertyTypeString = 0
PropertyTypeInt8 = 1
PropertyTypeInt16 = 2
PropertyTypeInt32 = 3
PropertyTypeInt64 = 4
PropertyTypeFloat32 = 5
PropertyTypeFloat64 = 6
PropertyTypeBool = 7
PropertyTypeBytes = 8
PropertyTypeNull = 9
```

These map directly to the types provided by the MQ series library used by the bridge.

<a name="body"></a>

### The Message Body

The message body in MQ series is mapped directly to a body field in the msgpack encoding.

<a name="reqrep"></a>

## Request-Reply

The bridge tries to respect request-reply semantics. When a message comes in from MQ series with the ReplyToQ header set the bridge will try to find an equivalent subject or channel. In the case of MQ-NATS the subject that maps to the ReplyToQ will be used as the reply to subject. In the case of streaming, a special field in the header ReplyToChannel will be set with the streaming version of the reply to.

Messages coming from streaming or NATS will have their ReplyToQ and ReplyToQMgr headers set before going into MQ series, if they have a reply to subject or the ReplyToChannel field set in the encoded message.

Keep in mind that this bi-directional request-reply support requires two connectors, one for MQ-NATS/STAN and one for NATS/STAN-MQ in the same bridge.

<a name="helpers"></a>

## Helpers

The bridge comes with a few helpers to read/write encoded messages.

<a name="golang"></a>

### Go

For go, the `/message` package in the bridge code contains everything needed to encode and decode messages.

Use the `NewBridgeMessage(body []byte)` function to create a message with a set of bytes for the body. Then use `SetProperty()` method on the message to set properties, or set header fields directly.

Use `DecodeBridgeMessage(encoded []byte)` to create a new message from an encoded set of bytes. Read the headers and body directly, or read properties with the various `Get*Property` methods.

<a name="java"></a>

### Java

A helper class for java is provided in the `/helpers/java` folder. This code depends on the jackson annotations, jackson object encoder and msg pack library. To create a new message, use `Message.NewMessageWithBody(byte[] body)` and then call `encode()` to get out the encoded bytes.

To decode a message use `Message.DecodeMessage([]byte encoded)` to create the message object. There is a `getHeader()` method to access the MQ header metadata and assorted `get*Property` method to get the properties.
