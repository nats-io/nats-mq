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

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

import org.junit.Test;

public class MessageTest {
    @Test
    public void testInterchange() throws IOException {
        Path path = Paths.get("../../resources", "interchange.bin");
        byte[] encoded = Files.readAllBytes(path);
        Message msg = Message.DecodeMessage(encoded);
        
        assertEquals("hello world", msg.getStringProperty("string"));

        assertEquals((byte)9, msg.getByteProperty("int8"));
        assertEquals((short)259, msg.getShortProperty("int16"));
        assertEquals((int)222222222, msg.getIntProperty("int32"));
        assertEquals(222222222222222222L, msg.getLongProperty("int64"));

        assertEquals((float)3.14, msg.getFloatProperty("float32"), 0.0001);
        assertEquals((double)6.4999, msg.getDoubleProperty("float64"), 0.0001);

        assertTrue(msg.getBooleanProperty("bool"));

        byte[] bytes = msg.getBytesProperty("bytes");
        String asString = new String(bytes, StandardCharsets.UTF_8);
        assertEquals("one two three four", asString);

        assertEquals("hello world", msg.getBodyAsUTF8String());
        assertEquals(1, msg.getHeader().getVersion());
        assertEquals(2, msg.getHeader().getReport());

        String id = new String(msg.getHeader().getMsgID(), StandardCharsets.UTF_8);
        assertEquals("cafebabe", id);
    }
}