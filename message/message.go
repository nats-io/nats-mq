package message

import (
	"fmt"
	"github.com/ugorji/go/codec"
	"reflect"
)

var mh codec.MsgpackHandle

func init() {
	mh.RawToString = true
	mh.WriteExt = true
	mh.SignedInteger = true
}

const (
	// PropertyTypeString type value for a string property
	PropertyTypeString = 0

	// PropertyTypeInt8 type value for an int8 property
	PropertyTypeInt8 = 1

	// PropertyTypeInt16 type value for an int16 property
	PropertyTypeInt16 = 2

	// PropertyTypeInt32 type value for an int32 property
	PropertyTypeInt32 = 3

	// PropertyTypeInt64 type value for an int64 property
	PropertyTypeInt64 = 4

	// PropertyTypeFloat32 type value for a float32 property
	PropertyTypeFloat32 = 5

	// PropertyTypeFloat64 type value for a float64 property
	PropertyTypeFloat64 = 6

	// PropertyTypeBool type value for a bool property
	PropertyTypeBool = 7

	// PropertyTypeBytes type value for a []byte property
	PropertyTypeBytes = 8

	// PropertyTypeNull type value for a null property
	PropertyTypeNull = 9
)

// BridgeHeader maps to an MQMD struct in the MQ messages
type BridgeHeader struct {
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

// Property wraps a typed property to allow proper round/trip support
// with MQ in the bridge
type Property struct {
	Type  int
	Value interface{}
}

// Properties is a wrapper for a map of named properties
type Properties map[string]Property

// BridgeMessage is the NATS-side wrapper for the mq message
type BridgeMessage struct {
	Header     BridgeHeader `codec:"header,omitempty"`
	Properties Properties   `codec:"props,omitempty"`
	Body       []byte       `codec:"body,omitempty"`
}

// NewBridgeMessage creates an empty message with the provided body, the header is empty
func NewBridgeMessage(body []byte) *BridgeMessage {
	return &BridgeMessage{
		Body:       body,
		Properties: Properties{},
	}
}

// DecodeBridgeMessage decodes the bytes and returns the decoded version
// use NewBridgeMessage to create a message with an empty header
func DecodeBridgeMessage(data []byte) (*BridgeMessage, error) {
	mqMsg := &BridgeMessage{}
	dec := codec.NewDecoderBytes(data, &mh)
	err := dec.Decode(mqMsg)

	if err != nil {
		return nil, err
	}

	return mqMsg, nil
}

// SetProperty adds a property to the map with the correct type, using reflection to get
// the type out of the value. An error is returned if the type isn't supported
// All unsigned ints (except uint8==byte) are converted to int64, as is int.
// A single byte is cast to an int8 to match the MQ series library
func (msg *BridgeMessage) SetProperty(name string, value interface{}) error {
	if value == nil {
		msg.Properties[name] = Property{
			Type:  PropertyTypeNull,
			Value: value,
		}
		return nil
	}

	switch value.(type) {
	case string:
		msg.Properties[name] = Property{
			Type:  PropertyTypeString,
			Value: value,
		}
	case bool:
		msg.Properties[name] = Property{
			Type:  PropertyTypeBool,
			Value: value,
		}
	case int8, byte:
		msg.Properties[name] = Property{
			Type:  PropertyTypeInt8,
			Value: value,
		}
	case int16:
		msg.Properties[name] = Property{
			Type:  PropertyTypeInt16,
			Value: value,
		}
	case int32:
		msg.Properties[name] = Property{
			Type:  PropertyTypeInt32,
			Value: value,
		}
	case int64, uint16, uint32, uint64, int:
		msg.Properties[name] = Property{
			Type:  PropertyTypeInt64,
			Value: value,
		}
	case float32:
		msg.Properties[name] = Property{
			Type:  PropertyTypeFloat32,
			Value: value,
		}
	case float64:
		msg.Properties[name] = Property{
			Type:  PropertyTypeFloat64,
			Value: value,
		}
	case []byte:
		msg.Properties[name] = Property{
			Type:  PropertyTypeBytes,
			Value: value,
		}
	default:
		return fmt.Errorf("can't set property of type %q", reflect.TypeOf(value))
	}
	return nil
}

// GetTypedProperty takes a property
func (msg *BridgeMessage) GetTypedProperty(name string) (interface{}, bool) {
	prop, ok := msg.Properties[name]

	if !ok {
		return nil, false
	}

	switch prop.Type {
	case PropertyTypeBool:
		return msg.GetBoolProperty(name)
	case PropertyTypeBytes:
		return msg.GetBytesProperty(name)
	case PropertyTypeInt8:
		return msg.GetInt8Property(name)
	case PropertyTypeInt16:
		return msg.GetInt16Property(name)
	case PropertyTypeInt32:
		return msg.GetInt32Property(name)
	case PropertyTypeInt64:
		return msg.GetInt64Property(name)
	case PropertyTypeFloat32:
		return msg.GetFloat32Property(name)
	case PropertyTypeFloat64:
		return msg.GetFloat64Property(name)
	case PropertyTypeString:
		return msg.GetStringProperty(name)
	case PropertyTypeNull:
		return nil, true
	}
	return nil, false
}

// GetStringProperty returns the property as a string and true, or  "" and false if the property
// is not a string, or doesn't exist.
func (msg *BridgeMessage) GetStringProperty(name string) (string, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeString {
		return prop.Value.(string), true
	}

	return "", false
}

func (msg *BridgeMessage) parseInt(v interface{}) (int64, bool) {
	i := 0
	switch v.(type) {
	case int:
		i = int(v.(int))
	case int8:
		i = int(v.(int8))
	case int16:
		i = int(v.(int16))
	case int32:
		i = int(v.(int32))
	case int64:
		i = int(v.(int64))
	default:
		return 0, false
	}
	return int64(i), true
}

// GetInt8Property returns the property as a sized int and true, or  0 and false if the property
// is not the correct size int, or doesn't exist.
func (msg *BridgeMessage) GetInt8Property(name string) (int8, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeInt8 {
		val, ok := msg.parseInt(prop.Value)
		if ok {
			return int8(val), true
		}
		return 0, false
	}

	return 0, false
}

// GetInt16Property returns the property as a sized int and true, or  0 and false if the property
// is not the correct size int, or doesn't exist.
func (msg *BridgeMessage) GetInt16Property(name string) (int16, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeInt16 {
		val, ok := msg.parseInt(prop.Value)
		if ok {
			return int16(val), true
		}
		return 0, false
	}

	return 0, false
}

// GetInt32Property returns the property as a sized int and true, or  0 and false if the property
// is not the correct size int, or doesn't exist.
func (msg *BridgeMessage) GetInt32Property(name string) (int32, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeInt32 {
		val, ok := msg.parseInt(prop.Value)
		if ok {
			return int32(val), true
		}
		return 0, false
	}

	return 0, false
}

// GetInt64Property returns the property as a sized int and true, or  0 and false if the property
// is not the correct size int, or doesn't exist.
func (msg *BridgeMessage) GetInt64Property(name string) (int64, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeInt64 {
		return msg.parseInt(prop.Value)
	}

	return 0, false
}

func (msg *BridgeMessage) parseFloat(v interface{}) (float64, bool) {
	i := 0.0
	switch v.(type) {
	case float32:
		i = float64(v.(float32))
	case float64:
		i = float64(v.(float64))
	default:
		return 0, false
	}
	return i, true
}

// GetFloat32Property returns the property as a sized float and true, or  0 and false if the property
// is not the correct size float, or doesn't exist.
func (msg *BridgeMessage) GetFloat32Property(name string) (float32, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeFloat32 {
		val, ok := msg.parseFloat(prop.Value)
		if ok {
			return float32(val), true
		}
	}

	return 0, false
}

// GetFloat64Property returns the property as a sized float and true, or  0 and false if the property
// is not the correct size float, or doesn't exist.
func (msg *BridgeMessage) GetFloat64Property(name string) (float64, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeFloat64 {
		return msg.parseFloat(prop.Value)
	}

	return 0, false
}

// GetBoolProperty returns the property as a boolean and true, or false and false if the property
// is not a bool, or doesn't exist.
func (msg *BridgeMessage) GetBoolProperty(name string) (bool, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeBool {
		return prop.Value.(bool), true
	}

	return false, false
}

// GetBytesProperty returns the property as a []byte and true, or nil and false if the property
// is not a []byte, or doesn't exist.
func (msg *BridgeMessage) GetBytesProperty(name string) ([]byte, bool) {
	prop, ok := msg.Properties[name]

	if ok && prop.Type == PropertyTypeBytes {
		return prop.Value.([]byte), true
	}

	return nil, false
}

// HasProperty returns true if the property exists
func (msg *BridgeMessage) HasProperty(name string) bool {
	_, ok := msg.Properties[name]
	return ok
}

// DeleteProperty removes a property from the map, and returns it
// If the item isn't present, nil is returned
func (msg *BridgeMessage) DeleteProperty(name string) interface{} {
	cur, ok := msg.Properties[name]

	delete(msg.Properties, name)

	if ok {
		return cur.Value
	}

	return nil
}

// Encode a bridge message to bytes
func (msg *BridgeMessage) Encode() ([]byte, error) {
	var raw [1024]byte
	b := raw[:0]
	enc := codec.NewEncoderBytes(&b, &mh)
	err := enc.Encode(msg)

	if err != nil {
		return nil, err
	}

	return b, nil
}
