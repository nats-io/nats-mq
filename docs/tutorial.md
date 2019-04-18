# NATS-MQ Tutorial

Bridges inherently have a lot of parts, let's walkthrough a setup with NATS, MQ series, the bridge, a GO client and a Java client. The code for the two clients is available in this repo under the `message/examples` and `java/src/main/java/io/nats/mqbridge/examples` folders. The high level architecture is this:

* The bridge will be configured to map:
  * The subject *tick* to the queue *DEV.QUEUE.1*
  * The queue *DEV.QUEUE.1* to the subject *tock*

* The `tick` example will send a message to NATS on the subject *tick*. This message contains
  * A user specified string
  * A counter property
  * A property with the current time as a unix timestamp
  * A correlation id

* Messages coming from *tick* will be bridged to *DEV.QUEUE.1* and then bridged back to the subject *tock*

* The `tock` example will subscribe to *tock* and print the contents of each message received

In order to run the tutorial, we will:

1. [Prepare your Terminal](#terminal)
1. [Run the NATS Server](#gnatsd)
1. [Run the MQ Server](#mqserver)
1. [Configure the Bridge](#config)
1. [Run the Bridge Locally](#localbridge)
 or [Run the Bridge with Docker](#dockerbridge)
1. [Check the Monitoring Page](#monitoring)
1. [Build and Run the Go Tock Example](#gotock)
1. [Build and Run the Java Tick Example](#javatick)
1. [Compare the Output](#output)
1. [Multiplex with Multiple Tickers](#both)

<a name="terminal"></a>

## 1. Prepare your Terminal

All of the commands in this tutorial assume that you are in the `nats-mq` repository folder, at the root.

```bash
% git clone https://github.com/nats-io/nats-mq.git
% cd nats-mq
```

Running the go and Java examples will require further setup described below.

You will likely want 5 shells open to make this easier.

This tutorial relies on docker to reduce dependency requirements, but you can run things directly by altering the bridge configuration and following the steps in [buildandrun.md](buildandrun.md) to build the bridge.

<a name="gnatsd"></a>

## 2. Run the NATS server

The nats-server for this tutorial doesn't need any special settings and can be run on the default port. Simply execute:

```bash
% gnatsd
[43179] 2019/04/18 10:08:25.758721 [INF] Starting nats-server version 2.0.0-RC5
[43179] 2019/04/18 10:08:25.758785 [INF] Git commit [not set]
[43179] 2019/04/18 10:08:25.759864 [INF] Listening for client connections on 0.0.0.0:4222
[43179] 2019/04/18 10:08:25.759870 [INF] Server id is NDOE767A6DCXXLLSDTLGKKBY5IJREWHITNEILOMTU5XPRWQJV6U5Y3WC
[43179] 2019/04/18 10:08:25.759872 [INF] Server is ready
```

> If you want to run the server elsewhere, be sure to update the bridge [configuration](config.md) to match.

<a name="mqserver"></a>

## 3. Run the MQ Server

See [https://hub.docker.com/r/ibmcom/mq/](https://hub.docker.com/r/ibmcom/mq/) for instructions on getting the docker image for the MQ series server. A script is provided to launch the docker image for the bridge in the `scripts` folder, but we can also run it manually using:

```bash
% docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  ibmcom/mq
2019-04-18T17:11:26.300Z Set password for "admin" user
2019-04-18T17:11:26.322Z Using queue manager name: QM1
...
2019-04-18T17:11:27.701Z AMQ5026I: The listener 'DEV.LISTENER.TCP' has started. ProcessId(482).
2019-04-18T17:11:37.119Z Started web server
```

This will create some default queues, in particular the *DEV.QUEUE.1* queue we will use in this tutorial.

> If you want to use a different MQ server or queue, you will need to update the bridge [configuration](config.md) appropriately.

<a name="config"></a>

## 4. Configure the Bridge

A copy of the configuration we will use is in resources/tiktok.conf:

```yaml
reconnectinterval: 5000,

logging: {
  Time: true,
  Debug: true,
  Trace: true,
  Colors: true,
  PID: false,
},

monitoring: {
  httpport: 9090,
}

nats: {
  Servers: ["localhost:4222"],
  ConnectTimeout: 5000,
  MaxReconnects: 5,
  ReconnectWait: 5000,
}

connect: [
  {
      type: NATS2Queue,
      id: "tick",
      mq: {
        ConnectionName: "localhost(1414)",
        ChannelName: "DEV.APP.SVRCONN",
        QueueManager: "QM1",
      },
      queue: "DEV.QUEUE.1",
      subject: "tick",
  },{
      type: Queue2NATS,
      id: "tock",
      mq: {
        ConnectionName: "localhost(1414)",
        ChannelName: "DEV.APP.SVRCONN",
        QueueManager: "QM1",
      },
      queue: "DEV.QUEUE.1",
      subject: "tock",
  },
],
```

This configuration:

* Sets the reconnect interval used by the bridge if it loses a connector.
* Turns on monitoring at port 9090
* Points NATS at the local server with a few default settings (no authentication)
* Creates two connectors:
  * "tick", of type *NATS2Queue* maps the NATS subject *tick* to the MQ queue *DEV.QUEUE.1*. The server is *localhost(1414)*, the channel is *DEV.APP.SVRCONN* and the queue manager is *QM1*.
  * "tock", of type *Queue2NATS* maps the MQ queue *DEV.QUEUE.1* to the NATS subject *tock*. The server is *localhost(1414)*, the channel is *DEV.APP.SVRCONN* and the queue manager is *QM1*.

We should see these two connectors come up when we run the bridge.

<a name="localbridge"></a>

## 5(a). Run the Bridge Locally

If you have [build](buildandrun.md) the bridge locally, you can run it with the tiktok.conf file. If you want to run with docker skip to [5(b)](#dockerbridge).

```bash
%  nats-mq -c resources/tiktok.conf
2019/04/18 10:35:45.128550 [INF] loading configuration from "resources/tiktok.conf"
2019/04/18 10:35:45.128925 [INF] starting MQ-NATS bridge, version 0.5
2019/04/18 10:35:45.128940 [INF] server time is Thu Apr 18 10:35:45 PDT 2019
2019/04/18 10:35:45.128947 [INF] connecting to NATS core
2019/04/18 10:35:45.132689 [INF] skipping NATS streaming connection, not configured
2019/04/18 10:35:45.132720 [TRC] starting connection NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 10:35:45.140536 [TRC] connected to queue manager QM1 at localhost(1414) as DEV.APP.SVRCONN for NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 10:35:45.141309 [TRC] opened and reading DEV.QUEUE.1
2019/04/18 10:35:45.141315 [INF] started connection NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 10:35:45.141321 [TRC] starting connection Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 10:35:45.145896 [TRC] connected to queue manager QM1 at localhost(1414) as DEV.APP.SVRCONN for Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 10:35:45.146671 [TRC] setting up callback for Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 10:35:45.146858 [TRC] opened and reading DEV.QUEUE.1
2019/04/18 10:35:45.146866 [INF] started connection Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 10:35:45.146945 [INF] starting http monitor on :9090
```

Notice that the connectors are created and the monitoring is enabled at port 9090. No NATS streaming was configured, so that connection was skipped.

<a name="dockerbridge"></a>

## 5(b). Run the Bridge with Docker

Rather than go through the full [build](buildandrun.md) process, you can use the docker image for the bridge.

First, we need to build the docker image:

```bash
% docker build -t "nats-io/mq-bridge:0.5" .
...
Successfully built c785b84f3b6d
Successfully tagged nats-io/mq-bridge:0.5
```

This will create an image with the tag `nats-io/mq-bridge` and the version `0.5`. Note the build can take a few minutes.

Once the image is complete we can run the bridge but we need a new configuration file that doesn't use *localhost*. In order to get the tiktok.conf to work on a mac, for example, you will need to replace:

localhost -> docker.for.mac.host.internal

A version of this file is available at `resources/mac.tiktok.conf`. If you are running on another OS, you may need to use a different mapping (please feel free to submit a PR with a config file for other operating systems.)

The bridge docker image expects the configuration to be available at `/mqbridge.conf` so we will map our example config to that location when we run docker. Also, the monitoring port in the config is set to 9090 so we will map that port out of the docker image:

```bash
 % docker run -v `pwd`/resources/mac.tiktok.conf:/mqbridge.conf -p 9090:9090 "nats-io/mq-bridge:0.5"
2019/04/18 17:46:10.458755 [INF] loading configuration from "/mqbridge.conf"
2019/04/18 17:46:10.461325 [INF] starting MQ-NATS bridge, version 0.5
2019/04/18 17:46:10.461364 [INF] server time is Thu Apr 18 17:46:10 UTC 2019
2019/04/18 17:46:10.461383 [INF] connecting to NATS core
2019/04/18 17:46:10.463342 [INF] skipping NATS streaming connection, not configured
2019/04/18 17:46:10.463391 [TRC] starting connection NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 17:46:10.474614 [TRC] connected to queue manager QM1 at docker.for.mac.host.internal(1414) as DEV.APP.SVRCONN for NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 17:46:10.475625 [TRC] opened and reading DEV.QUEUE.1
2019/04/18 17:46:10.475660 [INF] started connection NATS:tick to Queue:DEV.QUEUE.1
2019/04/18 17:46:10.475678 [TRC] starting connection Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 17:46:10.478857 [TRC] connected to queue manager QM1 at docker.for.mac.host.internal(1414) as DEV.APP.SVRCONN for Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 17:46:10.480211 [TRC] setting up callback for Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 17:46:10.480809 [TRC] opened and reading DEV.QUEUE.1
2019/04/18 17:46:10.481118 [INF] started connection Queue:DEV.QUEUE.1 to NATS:tock
2019/04/18 17:46:10.481274 [INF] starting http monitor on :9090
```

Notice that the connectors are created and the monitoring is enabled at port 9090. No NATS streaming was configured, so that connection was skipped.

<a name="monitoring"></a>

## 6. Check the Monitoring Page

You should be able to use your browser to check on the monitoring status for the bridge. Go to `http://localhost:9090`. There should be two links, one for varz and one for healthz. The healthz page will be empty, but shouldn't return an error. The varz page will show the connectors, with no data processed, see the bytes_in and bytes_out for each connector.

<a name="gotock"></a>

## 7. Build and Run the Go Tock Example

The Go example app should run without all the [MQ requirements](buildandrun.md), but it will download numerous go packages since it is part of the larger bridge repository. We will use `go run` to make sure that all the downloads take place. You can always build and install the example and run the executable directly.

The tock example doesn't listen to MQ-Series, it is listening to the *tock* subject which will be filled by the bridge based on messages coming in to a queue. It does use a decoder for msgpack, so you should download that package with:

```bash
% go get github.com/ugorji/go/codec
```

To run the tock listener, which will print messages from the tick example:

```bash
 % go run message/examples/tock/main.go
Listening on tock...
```

With this running, we are ready to start sending messages.

<a name="javatick"></a>

## 8. Build and Run the Java Tick Example

The Java helper library and tick/tock examples is in `helpers/java` and uses gradle to build and run. If you don't have gradle you will need to install it.

Once gradle is installed, you can build the library with:

```bash
% cd helpers/java
% gradle build
% gradle fatJar
```

This last command `gradle fatJar` grabs all the dependencies and puts them in 1 jar to make it easier to run the example.

Once built, you can run the example with the NATS server URL and your custom message:

```bash
% java -cp build/libs/mq-nats-1.0-fat.jar io.nats.mqbridge.examples.Tick nats://localhost:4222 "welcome to the nats-mq bridge"
```

The tick example sends a message every second. Let's look at the output...

<a name="output"></a>

## 9. Compare the Output

Tick will print each message as it sends it:

```bash
% java -cp build/libs/mq-nats-1.0-fat.jar io.nats.mqbridge.examples.Tick nats://localhost:4222 "welcome to the nats-mq bridge"
Sending message:
    body: welcome to the nats-mq bridge
    counter: 1
    time: Thu 4 18 11:18:54 PDT 2019
    id: 7KhSTITiw7K07QliBcti7K

Sending message:
    body: welcome to the nats-mq bridge
    counter: 2
    time: Thu 4 18 11:18:55 PDT 2019
    id: 7KhSTITiw7K07QliBctiAj
```

Tock will print each message as it receives it:

```bash
 % go run message/examples/tock/main.go
Listening on tock...

Received message:
    body: welcome to the nats-mq bridge
    counter: 1
    time: Thu 4 18 11:18:54 PDT 2019
    id: 7KhSTITiw7K07QliBcti7K

Received message:
    body: welcome to the nats-mq bridge
    counter: 2
    time: Thu 4 18 11:18:55 PDT 2019
    id: 7KhSTITiw7K07QliBctiAj
```

Messages have a unique counter, time and id. Each message will travel from tick to NATS to the bridge to MQ-Series to the bridge to NATS and finally to the tock example.

<a name="both"></a>

## 10. Multiplex with Multiple Tickers

A version of tick and tock are available in Java and go and the examples use core NATS so you can run both the Java and go versions at the same time and see the mix of messages from the Go ticker and the Java ticker:

```bash
Received message:
    body: go ticker
    counter: 4
    time: Thu Apr 18 11:23:23 PDT 2019
    id: JdclO7I1S2REvJfzYwoxVQ

Received message:
    body: java ticker
    counter: 6
    time: Thu 4 18 11:23:24 PDT 2019
    id: NDhC21HQEKU7JOzjU8WGh2
```