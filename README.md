# NATS-MQ Bridge

This project implements a simple, but generic, bridge between NATS or NATS streaming and MQ Series queues and topics.

## Features

* Support for bridging from/to MQ-Series queues or topics
* Arbitrary subjects in NATS, wildcards for incoming messages
* Arbitrary channels in NATS streaming
* Optional durable subscriber names for streaming
* Complete mapping with message headers, properties and the message body
* An option to only pass message bodies
* Request/Reply mapping, when connectors are available
* Configurable std-out logging
* A single configuration file, with support for reload
* Optional SSL to/from MQ-Series, NATS and NATS streaming
* HTTP/HTTPS-based monitoring endpoints for health or statistics

## Overview

The bridge runs as a single process with a configured set of connectors mapping an MQ-Series queue or topic to a NATS subject or a NATS streaming channel. Connectors can also map the opposite direction from NATS to MQ-Series. Each connector is a one-way bridge.

Connectors share a NATS connection and an optional connection to the NATS streaming server. **Connectors each create a connection to the MQ server, subject to TCP connection sharing in the underlying library**

Messages can be forwarded with or without headers. This mapping is as bi-directional as possible. NATS clients can send messages with MQ headers set, and NATS clients can read the headers contained in MQ messages. However, there are a few limitations where NATS to MQ messages will have headers stripped because they can't be passed in to the queue or topic. When headers are included, the contents of the NATS message is prescribed by a [msgpack-based format.](docs/messages.md) Connectors set to exclude headers will just use the body of the MQ message as the entire NATS message.

Request-reply is supported for Queues. Topics with a reply-to queue should work, but reply-to topics are not supported. This support is based on the bridge's configuration. If the bridge maps `queue1` to `subject1` and `subject2` to `queue2`, then a message to `queue1` with a reply-to queue of `queue2` will go out on NATS `subject1` with a reply-to of `subject2`, and vice-versa for the other direction. **Request-reply requires message headers**, and will not work if headers are excluded.

The bridge is [configured with a NATS server-like format](docs/config.md), in a single file and uses the NATS logger.

An [optional HTTP/HTTPS endpoint](docs/monitoring.md) can be used for monitoring.

## Documentation

* [Build & Run the Bridge](docs/buildandrun.md)
* [Configuration](docs/config.md)
* [Message Format](docs/messages.md)
* [Monitoring](docs/monitoring.md)

## External Resources

* [NATS](https://nats.io/documentation/)
* [NATS server](https://github.com/nats-io/gnatsd)
* [NATS Streaming](https://github.com/nats-io/nats-streaming-server)
* [MQ Library](https://github.com/ibm-messaging/mq-golang)

## License

Unless otherwise noted, the NATS-MQ bridge source files are distributed under the Apache Version 2.0 license found in the LICENSE file.