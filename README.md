# nats-mq

Simple bridge between NATS streaming and MQ Series

## Developing

### The MQSeries library

The go [mq series library](https://github.com/ibm-messaging/mq-golang) requires the client libraries. These are referenced from the readme, except for [MacOS which are available here](https://developer.ibm.com/messaging/2019/02/05/ibm-mq-macos-toolkit-for-developers/).

```bash
% export MQ_INSTALLATION_PATH=<your installation library>
% export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
% export CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"
% export CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
 ```

 *Note there is a typo on the web page, missing `-rpath` and has `rpath` instead.

 Build the MQ library:

 ```bash
 go install ./ibmmq
 go install ./mqmetric
 ```

 you may see `ld: warning: directory not found for option '-L/opt/mqm/lib64'` but you can ignore it.

 You will also need to set these variables for building the bridge itself, since it depends on the MQ series packages.

 The dependency on the MQ package requires v3.3.4 to fix an rpath issue on Darwin.

### Running the examples from the Go library

The examples that pub/sub require an environment that tells them where the server is. You can use something like:

```bash
% export MQSERVER="DEV.APP.SVRCONN/TCP/localhost(1414)"
```

for the default docker setup described below. This will allow you to run examples:

```bash
% go run amqsput.go DEV.QUEUE.1 QM1
```

### Running the docker container for MQ Series

See [https://hub.docker.com/r/ibmcom/mq/](https://hub.docker.com/r/ibmcom/mq/) to get the docker container.

Also check out the [usage documentation](https://github.com/ibm-messaging/mq-container/blob/master/docs/usage.md).

```bash
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  ibmcom/mq
```

or use the `scripts/run_mq.sh` to execute that command.

### Connecting to the docker web admin

[https://localhost:9443/ibmmq/console/](https://localhost:9443/ibmmq/console/)

The login is:

```bash
User: admin
Password: passw0rd
```

Chrome may complain because the certificate for the server isn't valid, but go on through.

### Connecting with an application

For applications the login is documented as:

```bash
User: app
Password: *none*
```

I found that simply turning off user name and password works.