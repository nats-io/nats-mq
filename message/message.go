// Copyright 2012-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package message

import (
	"fmt"
	"reflect"

	"github.com/ugorji/go/codec"
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

// Property wraps a typed property to allow proper round/trip support
// with MQ in the bridge
type Property struct {
	Type  int         `codec:"type,omitempty"`
	Value interface{} `codec:"value,omitempty"`
}

// Properties is a wrapper for a map of named properties
type Properties map[string]Property

// BridgeMessage is the NATS-side wrapper for the mq message
type BridgeMessage struct {
	Body       []byte       `codec:"body,omitempty"`
	Header     BridgeHeader `codec:"header,omitempty"`
	Properties Properties   `codec:"props,omitempty"`
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
	if data == nil {
		return nil, fmt.Errorf("attempt to decode bridge message of zero length")
	}

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
		switch prop.Value.(type) {
		case string:
			return prop.Value.(string), true
		default:
			return "", false
		}
	}

	return "", false
}

func (msg *BridgeMessage) parseInt(v interface{}) (int64, bool) {
	i := 0
	switch v := v.(type) {
	case int:
		i = int(v)
	case int8:
		i = int(v)
	case int16:
		i = int(v)
	case int32:
		i = int(v)
	case int64:
		i = int(v)
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
	switch v := v.(type) {
	case float32:
		i = float64(v)
	case float64:
		i = float64(v)
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
