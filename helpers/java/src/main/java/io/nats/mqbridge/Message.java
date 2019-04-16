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
package io.nats.mqbridge;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.NoSuchElementException;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;

import org.msgpack.jackson.dataformat.MessagePackFactory;

public class Message {
    static class Property {
        static final int PROPERTY_TYPE_STRING = 0;
        static final int PROPERTY_TYPE_INT8 = 1;
        static final int PROPERTY_TYPE_INT16 = 2;
        static final int PROPERTY_TYPE_INT32 = 3;
        static final int PROPERTY_TYPE_INT64 = 4;
        static final int PROPERTY_TYPE_FLOAT32 = 5;
        static final int PROPERTY_TYPE_FLOAT64 = 6;
        static final int PROPERTY_TYPE_BOOL = 7;
        static final int PROPERTY_TYPE_BYTES = 8;
        static final int PROPERTY_TYPE_NULL = 9;

        @JsonProperty("type")
        int type;

        @JsonProperty("value")
        Object value;

        Property(){}

        @JsonCreator
        Property(@JsonProperty("type") int type, @JsonProperty("value") Object value) {
            this.type = type;

            if (value instanceof Integer) {
                value = new Long(((Integer)value).longValue());
            }

            if (value instanceof Float) {
                value = new Double(((Float)value).doubleValue());
            }
            this.value = value;
        }
    }

    @JsonProperty("header")
    private Header header;

    @JsonProperty("props")
    private HashMap<String, Property> properties;

    @JsonProperty("body")
    private byte[] body;

    public static Message NewMessageWithBody(byte[] body) {
        Message msg = new Message();
        msg.body = body;
        return msg;
    }

    public static Message DecodeMessage(byte[] encoded) throws JsonParseException, JsonMappingException, IOException {
        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        return objectMapper.readValue(encoded, Message.class);
    }

    private Message() {
        this.properties = new HashMap<>();
        this.header = new Header();
        this.body = null;
    }

    /**
     * @return the body or null
     */
    public byte[] getBody() {
        return body;
    }

    /**
     * Tries to convert the body to a UTF-8 string and return it.
     * @return the string or "" if the body is null
     */
    public String getBodyAsUTF8String() {
        if (this.body == null) {
            return "";
        }

        return new String(this.body, StandardCharsets.UTF_8);
    }

    public Header getHeader() {
        return header;
    }

    public byte[] encode() throws IOException {
        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        return objectMapper.writeValueAsBytes(this);
    }

    /**
     * @param the name of the property
     * @return true if the property exists, no type information is checked
     */
    public boolean hasProperty(String name) {
        return this.properties.containsKey(name);
    }

    /**
     * @param the name of the property
     * @return true if the property existed
     */
    public boolean deleteProperty(String name) {
        return this.properties.remove(name) != null;
    }

    /**
     * Store a string property.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setStringProperty(String name, String value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_STRING;
        prop.value = value;
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the string property
     * @throws NoSuchElementException if the  property doesn't exist or is not a string
     */
    public String getStringProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_STRING) {
            throw new NoSuchElementException();
        }

        return (String) prop.value;
    }

    /**
     * Store an integer-type property, internally the data is stored in a Long with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setByteProperty(String name, byte value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_INT8;
        prop.value = new Long(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the byte property
     * @throws NoSuchElementException if the  property doesn't exist or is not a byte (int8)
     */
    public byte getByteProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_INT8) {
            throw new NoSuchElementException();
        }

        return ((Long)prop.value).byteValue();
    }

    /**
     * Store an integer-type property, internally the data is stored in a Long with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setShortProperty(String name, byte value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_INT16;
        prop.value = new Long(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the short property
     * @throws NoSuchElementException if the  property doesn't exist or is not a short (int16)
     */
    public short getShortProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_INT16) {
            throw new NoSuchElementException();
        }

        return ((Long)prop.value).shortValue();
    }

    /**
     * Store an integer-type property, internally the data is stored in a Long with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setIntProperty(String name, int value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_INT32;
        prop.value = new Long(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the integer property
     * @throws NoSuchElementException if the  property doesn't exist or is not an integer (int32)
     */
    public int getIntProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_INT32) {
            throw new NoSuchElementException();
        }

        return ((Long)prop.value).intValue();
    }

    /**
     * Store an integer-type property, internally the data is stored in a Long with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setLongProperty(String name, long value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_INT64;
        prop.value = new Long(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the long property
     * @throws NoSuchElementException if the  property doesn't exist or is not a long (int64)
     */
    public long getLongProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_INT64) {
            throw new NoSuchElementException();
        }

        return ((Long)prop.value).longValue();
    }

    /**
     * Store an float-type property, internally the data is stored in a Double with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setFloatProperty(String name, float value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_FLOAT32;
        prop.value = new Double(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the float property
     * @throws NoSuchElementException if the  property doesn't exist or is not a float (float32)
     */
    public float getFloatProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_FLOAT32) {
            throw new NoSuchElementException();
        }

        return ((Double)prop.value).floatValue();
    }

    /**
     * Store an float-type property, internally the data is stored in a Double with a
     * type attached.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setDoubleProperty(String name, float value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_FLOAT64;
        prop.value = new Double(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the double property
     * @throws NoSuchElementException if the  property doesn't exist or is not a double (float64)
     */
    public double getDoubleProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_FLOAT64) {
            throw new NoSuchElementException();
        }

        return ((Double)prop.value).doubleValue();
    }

    /**
     * Store a boolean property.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setBooleanProperty(String name, boolean value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_BOOL;
        prop.value = Boolean.valueOf(value);
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the boolean property
     * @throws NoSuchElementException if the  property doesn't exist or is not a boolean
     */
    public boolean getBooleanProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_BOOL) {
            throw new NoSuchElementException();
        }

        return ((Boolean)prop.value).booleanValue();
    }

    /**
     * Store a byte array property. DOES NOT COPY the bytes.
     * @param name the name of the property
     * @param value the value of the property
     */
    public void setBytesProperty(String name, byte[] value) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_BYTES;
        prop.value = value;
        this.properties.put(name, prop);
    }

    /**
     * @param the name of the property
     * @return the byte array property
     * @throws NoSuchElementException if the  property doesn't exist or is not a boolean
     */
    public byte[] getBytesProperty(String name) throws NoSuchElementException {
        if (!this.hasProperty(name)) {
            throw new NoSuchElementException();
        }

        Property prop = this.properties.get(name);
        if (prop.type != Property.PROPERTY_TYPE_BYTES) {
            throw new NoSuchElementException();
        }

        return (byte[]) prop.value;
    }

    /**
     * Store a null property. Use hasProperty to check for null properties.
     * @param name the name of the property
     */
    public void setNullProperty(String name) {
        Property prop = new Property();
        prop.type = Property.PROPERTY_TYPE_NULL;
        this.properties.put(name, prop);
    }
}