package message

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	msg := NewBridgeMessage([]byte("hello world"))
	msg.Header = BridgeHeader{
		Version: 1,
		Report:  2,
	}

	encoded, err := msg.Encode()
	require.NoError(t, err)

	copy, err := DecodeBridgeMessage(encoded)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte("hello world"), copy.Body)
	require.ElementsMatch(t, msg.Body, copy.Body)

	require.Equal(t, msg.Header.Version, copy.Header.Version)
	require.Equal(t, msg.Header.Report, copy.Header.Report)
}

func TestBadDecode(t *testing.T) {
	_, err := DecodeBridgeMessage([]byte("hello world"))
	require.Error(t, err)
}

func TestNilDecode(t *testing.T) {
	_, err := DecodeBridgeMessage(nil)
	require.Error(t, err)
}

func TestPropertyTypes(t *testing.T) {
	msg := NewBridgeMessage([]byte("hello world"))

	expected := map[string]interface{}{
		"string":  "hello world",
		"int8":    int8(9),
		"int16":   int16(259),
		"int32":   int32(222222222),
		"int64":   int64(222222222222222222),
		"float32": float32(3.14),
		"float64": float64(6.4999),
		"bool":    true,
		"bytes":   []byte("one two three four"),
	}

	for k, v := range expected {
		err := msg.SetProperty(k, v)
		require.NoError(t, err)

		actual, ok := msg.GetTypedProperty(k)
		require.True(t, ok)
		require.Equal(t, v, actual)
	}

	var actual interface{}

	key := "string"
	actual, ok := msg.GetStringProperty(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetStringProperty("bad")
	require.False(t, ok)

	key = "int8"
	actual, ok = msg.GetInt8Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetInt8Property("bad")
	require.False(t, ok)

	key = "int16"
	actual, ok = msg.GetInt16Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetInt16Property("bad")
	require.False(t, ok)

	key = "int32"
	actual, ok = msg.GetInt32Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetInt32Property("bad")
	require.False(t, ok)

	key = "int64"
	actual, ok = msg.GetInt64Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetInt64Property("bad")
	require.False(t, ok)

	key = "float32"
	actual, ok = msg.GetFloat32Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetFloat32Property("bad")
	require.False(t, ok)

	key = "float64"
	actual, ok = msg.GetFloat64Property(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetFloat64Property("bad")
	require.False(t, ok)

	key = "bool"
	actual, ok = msg.GetBoolProperty(key)
	require.True(t, ok)
	require.Equal(t, expected[key], actual)
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetBoolProperty("bad")
	require.False(t, ok)

	key = "bytes"
	actual, ok = msg.GetBytesProperty(key)
	require.True(t, ok)
	require.ElementsMatch(t, expected[key].([]byte), actual.([]byte))
	ok = msg.HasProperty(key)
	require.True(t, ok)
	actual, ok = msg.GetBytesProperty("bad")
	require.False(t, ok)

	encoded, err := msg.Encode()
	require.NoError(t, err)

	copy, err := DecodeBridgeMessage(encoded)
	require.NoError(t, err)

	// Props should match
	for k, v := range expected {
		actual, ok := copy.GetTypedProperty(k)
		require.True(t, ok)
		require.Equal(t, v, actual)
	}
}

func TestIntPropertyIs64Bit(t *testing.T) {
	msg := NewBridgeMessage(nil)
	err := msg.SetProperty("test", int(3333))
	require.NoError(t, err)

	actual, ok := msg.GetTypedProperty("test")
	require.True(t, ok)
	require.Equal(t, int64(3333), actual)

	ok = msg.HasProperty("test")
	require.True(t, ok)
}

func TestNullProperty(t *testing.T) {
	msg := NewBridgeMessage(nil)
	err := msg.SetProperty("test", nil)
	require.NoError(t, err)

	actual, ok := msg.GetTypedProperty("test")
	require.True(t, ok)
	require.Nil(t, actual)

	ok = msg.HasProperty("test")
	require.True(t, ok)

	_, ok = msg.GetTypedProperty("bad")
	require.False(t, ok)
}

func TestDeleteProperty(t *testing.T) {
	msg := NewBridgeMessage(nil)
	err := msg.SetProperty("test", "hello")
	require.NoError(t, err)

	actual, ok := msg.GetTypedProperty("test")
	require.True(t, ok)
	require.NotNil(t, actual)

	ok = msg.HasProperty("test")
	require.True(t, ok)

	old := msg.DeleteProperty("test")
	require.Equal(t, old, "hello")

	ok = msg.HasProperty("test")
	require.False(t, ok)

	old = msg.DeleteProperty("test")
	require.Nil(t, old)
}

func TestMismatchProperty(t *testing.T) {
	msg := NewBridgeMessage(nil)
	err := msg.SetProperty("test", "hello")
	require.NoError(t, err)

	actual, ok := msg.GetInt32Property("test")
	require.False(t, ok)
	require.Equal(t, actual, int32(0))
}

func TestUnknownType(t *testing.T) {
	msg := NewBridgeMessage(nil)
	err := msg.SetProperty("test", []string{"hello", "world"})
	require.Error(t, err)
}

func TestUnknownTypeOnGet(t *testing.T) {
	msg := NewBridgeMessage(nil)
	msg.Properties["test"] = Property{
		Type:  -1,
		Value: []int{1, 2, 3},
	}
	actual, ok := msg.GetTypedProperty("test")
	require.False(t, ok)
	require.Nil(t, actual)
}

func TestBadPropertyValues(t *testing.T) {
	msg := NewBridgeMessage(nil)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeString,
		Value: -1,
	}

	_, ok := msg.GetStringProperty("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeInt8,
		Value: "hello",
	}
	_, ok = msg.GetInt8Property("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeInt16,
		Value: "hello",
	}
	_, ok = msg.GetInt16Property("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeInt32,
		Value: "hello",
	}
	_, ok = msg.GetInt32Property("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeInt64,
		Value: "hello",
	}
	_, ok = msg.GetInt64Property("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeFloat32,
		Value: "hello",
	}
	_, ok = msg.GetFloat32Property("test")
	require.False(t, ok)

	msg.Properties["test"] = Property{
		Type:  PropertyTypeFloat64,
		Value: "hello",
	}
	_, ok = msg.GetFloat64Property("test")
	require.False(t, ok)
}

func TestMessageInterchange(t *testing.T) {

	encoded, err := ioutil.ReadFile("../resources/interchange.bin")
	require.NoError(t, err)

	msg, err := DecodeBridgeMessage(encoded)
	require.NoError(t, err)

	expected := map[string]interface{}{
		"string":  "hello world",
		"int8":    int8(9),
		"int16":   int16(259),
		"int32":   int32(222222222),
		"int64":   int64(222222222222222222),
		"float32": float32(3.14),
		"float64": float64(6.4999),
		"bool":    true,
		"bytes":   []byte("one two three four"),
	}

	for k, v := range expected {
		actual, ok := msg.GetTypedProperty(k)
		require.True(t, ok)
		require.Equal(t, v, actual)
	}

	require.Equal(t, "hello world", string(msg.Body))
	require.Equal(t, int32(1), msg.Header.Version)
	require.Equal(t, int32(2), msg.Header.Report)
	require.Equal(t, "cafebabe", string(msg.Header.MsgID))
}
