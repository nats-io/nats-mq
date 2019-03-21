# NATS-MQ Bridge Configuration

The bridge uses a single configuration file passed on the command line or environment variable. Configuration is organized into a root section and several blocks.

* [Specifying the Configuration File](#specify)
* [Shared](#root)
* [TLS](#tls)
* [Monitoring](#monitoring)
* [NATS](#nats)
* [NATS Streaming](#stan)
* [MQ Series](#mq)
* [Connectors](#connectors)

The configuration file format matches the NATS server and supports file includes of the form:

```yaml
include "./includes/connectors.conf"
```

<a name="specify"></a>

## Specifying the Configuration File

To set the configuration on the command line, use:

```bash
% nats-mq -c <path to config file>
```

To set the configuration file using an environment variable, export `MQNATS_BRIDGE_CONFIG` with the path to the configuration.

<a name="root"></a>

## Root Section

The root section:

```yaml
reconnectinterval: 5000,
```

can currently contain settings for:

* `reconnectinterval` - this value, in milliseconds, is the time used in between reconnection attempts for a connector when it fails. For example, if a connector loses access to NATS, the bridge will try to restart it every `reconnectinterval` milliseconds.

## TLS <a name="tls"></a>

NATS, streaming and HTTP configurations take an optional TLS setting. The TLS configuration takes three possible settings:

* `root` - file path to a CA root certificate store, used for NATS connections
* `cert` - file path to a server certificate, used for HTTPS monitoring and optionally for client side certificates with NATS
* `key` - key for the certificate store specified in cert

MQ SSL properties are configured via a different structure discussed [below](#mq).

<a name="monitoring"></a>

## Monitoring

The monitoring section:

```yaml
monitoring: {
  httpsport: -1,
  tls: {
      cert: /a/server-cert.pem,
      key: /a/server-key.pem,
  }
}
```

Is used to configure an HTTP or HTTPS port, as well as TLS settings when HTTPS is used.

* `httphost` - the network interface to publish monitoring on, valid for HTTP or HTTPS. An empty value will tell the server to use all available network interfaces.
* `httpport` - the port for HTTP monitoring, no TLS configuration is expected, a value of -1 will tell the server to use an ephemeral port, the port will be logged on startup.

`2019/03/20 12:06:38.027822 [INF] starting http monitor on :59744`

* `httpsport` - the port for HTTPS monitoring, a TLS configuration is expected, a value of -1 will tell the server to use an ephemeral port, the port will be logged on startup.
* `tls` - a [TLS configuration](#tls).

The `httpport` and `httpsport` settings are mutually exclusive, if both are set to a non-zero value the bridge will not start.

<a name="nats"></a>

## NATS

The bridge makes a single connection to NATS. This connection is shared by all connectors. Configuration is through the `nats` section of the config file:

```yaml
nats: {
  Servers: ["localhost:4222"],
  ConnectTimeout: 5000,
  MaxReconnects: 5,
  ReconnectWait: 5000,
}
```

NATS can be configured with the following properties:

* `servers` - an array of server URLS
* `connecttimeout` - the time, in milliseconds, to wait before failing to connect to the NATS server
* `reconnectwait` - the time, in milliseconds, to wait between reconnect attempts
* `maxreconnects` - the maximum number of reconnects to try before exiting the bridge with an error.
* `tls` - (optional) [TLS configuration](#tls). If the NATS server uses unverified TLS with a valid certificate, this setting isn't required.
* `username` - (optional)  depending on the NATS server configuration, user name for authentication.
* `password` - (optional)  depending on the NATS server configuration, password for authentication.

<a name="stan"></a>

## NATS Streaming

The bridge makes a single connection to a NATS streaming server. This connection is shared by all connectors. Configuration is through the `stan` section of the config file:

```yaml
stan: {
  ClusterID: "test-cluster"
  ClientID: "mqbridge"
}
```

NATS streaming can be configured with the following properties:

* `clusterid` - the cluster id for the NATS streaming server.
* `clientid` - the client id for the bridge, shared by all connections.
* `pubackwait` - the time, in milliseconds, to wait before a publish fails due to a timeout.
* `discoverprefix` - the discover prefix for the streaming server.
* `maxpubacksinflight` - maximum pub ACK messages that can be in flight for this connection.
* `connectwait` - the time, in milliseconds, to wait before failing to connect to the streaming server.

<a name="mq"></a>

## MQ Series

Each connector in the bridge opens a connection, using `connx`, to the queue manager. These connection may share a TCP channel via the channel name. *The nats-mq bridge uses separate connections to implement transaction isolation.* Each connector will have an `mq` section in its configuration:

```yaml
mq: {
    ConnectionName: "localhost(1414)",
    ChannelName: "DEV.APP.SVRCONN",
    QueueManager: "QM1",
    UserName: "",
    Password: "",
},
```

The `mq` section supports the following properties:

* `connectionname` - the connection name, in MQ-speak, that specifies the server and port in the form `serverhost(port)`
* `channelname` - the connection channel name, may effect TCP sharing based on server configuration.
* `queuemanager` - the queue manager name.
* `username` - (optional) the username for connecting to the server.
* `password` - (optional) the password for connecting to the server.

as well as three SSL/TLS related properties:

* `keyrepository` - the path to a key file pair, for example, /a/key should result in a file `/a/key.kdb` and `/a/key.sth`.
* `certificatelabel` - (optional) the label of the certificate to use in the key repository.
* `sslpeername` - (optional) the peer name that should be in the server certificate.

<a name="connectors"></a>

## Connectors

The final piece of the bridge configuration is the `connect` section. Connect specifies an array of connector configurations. All connector configs use the same format, relying on optional settings to determine what the do. Each connector config also contains an [MQ config](#mq).

```yaml
connect: [
  {
      type: Queue2NATS,
      id: "alpha",
      mq: {
        ConnectionName: "localhost(1414)",
        ChannelName: "DEV.APP.SVRCONN",
        QueueManager: "QM1",
        UserName: "",
        Password: "",
      },
      queue: "DEV.QUEUE.1",
      subject: "one",
      excludeheaders: true,
  },{
      type: Queue2NATS,
      mq: {
        ConnectionName: "localhost(1414)",
        ChannelName: "DEV.APP.SVRCONN",
        QueueManager: "QM1",
        UserName: "",
        Password: "",
      },
      queue: "DEV.QUEUE.2",
      subject: "two",
      excludeheaders: true,
  },
],
```

The most important property in the connector configuration is the `type`. The type determines which kind of connector is instantiated. Available, uni-directional, types include:

* `Queue2NATS` - a queue to NATS connector
* `Queue2Stan` - a queue to streaming connector
* `NATS2Queue` - a NATS to queue connector
* `Stan2Queue` - a streaming to queue connector
* `Topic2NATS` - a topic to NATS connector
* `Topic2Stan` - a topic to streaming connector
* `NATS2Topic` - a NATS to topic connector
* `Stan2Topic` - a streaming to topic connector

There are two more properties that are used for all connectors. The first is used to specify if headers are mapped when coming from MQ or going to MQ. NATS messages going to the bridge must be [formatted correctly](messages.md) for this setting to work. NATS messages coming out of the bridge will be formatted automatically.

* `excludeheaders` - (optional) tells the bridge to skip message encoding and only send raw message bodies. The default is `false` which means that messages are encoded.

The second is an optional id, which is used in monitoring:

* `id` - (optional) user defined id that will tag the connection in monitoring JSON.

The remaining properties for a connector can be split by the type of connector used. To specify the MQ target/source, use either `topic` or `queue` along with the `mq` settings:

* `topic` - (exclusive with queue) the Topic to connect to
* `queue` - (exclusive with topic) the Queue to connect to

For NATS connections, specify:

* `subject` - the subject to subscribe/publish to, depending on the connections direction.

For streaming connections, there is a single required setting and several optional ones:

* `channel` - the streaming channel to subscribe/publish to.
* `durablename` - (optional) durable name for the streaming subscription (if appropriate.)
* `startatsequence` - (optional) start position, use -1 for start with last received, 0 for deliver all available (the default.)
* `startattime` - (optional) the start position as a time, in Unix seconds since the epoch, mutually exclusive with `startatsequence`.
