# Building and Running the NATS-MQ Bridge

The nats-mq bridge is written in GO and uses the [IBM library for MQ-Series](`github.com/ibm-messaging/mq-golang`). This library uses `cgo` to build on the MQI libraries. As a result the MQ series libraries are required on any machine used to build the bridge. As a result, the bridge is not released as an executable, rather users must build it themselves.

* [Running the Bridge](#run)
* [Building the Executable](#build)
  * [Setting up the MQSeries library](#mqlib)
  * [Running the library examples](#examples)
* [Testing the Bridge](#testing)
  * [Running the docker container for MQ Series](#docker)
  * [Connecting to the docker web admin](#web)
  * [Connecting with an application](#app)
  * [Using TLS](#tls)
* [Developer Notes](#developer)

<a name="run"></a>

## Running the Bridge

Once built, the bridge can be run as easily as any other executable, and [relies on a single flag](config.md#specify) to access the configuration file.

To run during development, you can create a configuration file and use `go run`:

```bash
%  go run nats-mq/main.go -c ~/Desktop/mqbridge.conf
```

Once built, the executable will be named nats-mq and can be run with a similar syntax:

```bash
% nats-mq -c ~/Desktop/mqbridge.conf
```

<a name="build"></a>

## Building the Executable

The nats-mq bridge is written in GO and uses the [IBM library for MQ-Series](`github.com/ibm-messaging/mq-golang`). This library uses `cgo` to build on the MQI libraries. As a result the MQ series libraries are required on any machine used to build the bridge.

> There is a fix in the mq-golang library version `7486f4a0b63560e3d0fdcd084b7c0d52b783dc33` that is required for integer properties.
> There is a fix in the mq-golang library version `c8adfe8` that is required for mq callbacks to work (currently in release 4.0.2 99a6892).

The dependency on the MQ package requires v3.3.4 to fix an rpath issue on Darwin. The commit listed above is past this version.

The bridge is implemented as a go module, all of the dependencies are specified in the go.mod file. For testing, the bridge embeds the nats-streaming-server which brings in a fair number of dependencies. The bridge executable only requires the nats and streaming clients as well as the mq-series library.

Once you have the dependencies in place, you can use the provided Makefile to build the bridge:

```bash
% make compile
```

This will build the bridge, but not install it. The compile task will also check formatting and imports. You can use `go install` to install the bridge in your GOPATH.

```bash
% go install ./...
```

<a name="mqlib"></a>

### Setting up the MQSeries library

The go [mq series library](https://github.com/ibm-messaging/mq-golang) requires the client libraries. These are referenced from the readme, except for [MacOS which are available here](https://developer.ibm.com/messaging/2019/02/05/ibm-mq-macos-toolkit-for-developers/).

Once you have the libraries installed, you need to set up your environment to point to them.

```bash
export MQ_INSTALLATION_PATH=<your installation library>
export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
export CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"
export CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
 ```

You can build the library packages using `go install`:

 ```bash
 go install ./ibmmq
 go install ./mqmetric
 ```

 you may see `ld: warning: directory not found for option '-L/opt/mqm/lib64'` but you can ignore it. If you can build the mq library, then your environment should be set up to build the bridge, as these variables are required for building the bridge as well.

<a name="examples"></a>

### Running the Library Examples

The go library contains a number of examples that you can use to test your MQ series connectivity. Most of these require an environment variable that tells them where the server is:

```bash
% export MQSERVER="DEV.APP.SVRCONN/TCP/localhost(1414)"
```

for the default docker setup described below. This will allow you to run examples:

```bash
% go run amqsput.go DEV.QUEUE.1 QM1
```

<a name="testing"></a>

### Testing the Bridge

The bridge tests are written to use docker to run the MQ server. You can use this docker container for developer testing as well. Once you have the docker image installed, you can run the tests with:

```bash
% make test
```

or

```bash
% go test ./...
```

<a name="docker"></a>

#### Running the Docker Image for MQ Series

See [https://hub.docker.com/r/ibmcom/mq/](https://hub.docker.com/r/ibmcom/mq/) for instructions on getting the docker image for the MQ series server. There is information in the [usage documentation](https://github.com/ibm-messaging/mq-container/blob/master/docs/usage.md) about running the container. The repository contains other documentation as well that may be helpful. We have tried to capture the main points here.

A simple example for running the docker image is:

```bash
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  ibmcom/mq
```

The bridge's `scripts` folder contains scripts to run the docker image.

* `scripts/run_mq.sh` - runs the docker image without detaching
* `scripts/run_mq_detached.sh` - runs the docker image detached
* `scripts/run_mq_tls.sh` - runs the docker image with the TLS certificates provided in the `/resources` folder.

<a name="web"></a>

#### Connecting to the Web Admin

If you need to connect to the web admin for the docker image. Use [https://localhost:9443/ibmmq/console/](https://localhost:9443/ibmmq/console/). The default login is:

```bash
User: admin
Password: passw0rd
```

Chrome may complain because the certificate for the server isn't valid, but go on through.

<a name="app"></a>

#### Connecting With an Application

For applications the login is documented as:

```bash
User: app
Password: *none*
```

We found that simply turning off user name and password works.

<a name="tls"></a>

#### TLS Setup

See [this ibm post](https://developer.ibm.com/messaging/learn-mq/mq-tutorials/secure-mq-tls/) for information about how the test TLS files for the docker image were created. The generated certs/keys are in the `resources` folder under `mqm`. A script is provided `scripts/run_mq_tls.sh` to run the server with this TLS setting. The TLS script will also set the app password to `passw0rd` for testing.

The server cert has the password `k3ypassw0rd`.

The client cert has the password `tru5tpassw0rd`, and the label `QM1.cert`

We created the kdb file using `runmqakm -cert -export -db client_key.p12 -pw tru5tpassw0rd -target_stashed -target_type kdb -target client.kdb -label "QM1.cert"`.

<a name="developer"></a>

### Developer notes

* The MQ series code can use callbacks or polling with get, to receive messages from Queues and Topics. There have been issues in the library for each so by having both the configuration can be used to pick one that works for the current setup.
* Using docker for tests will eat up docker space, you may need to run `docker system prune` once in a while to clean this up. The symptom of a full cache will be that the tests take forever to run because they fail to run the MQ series server image and spend 30s trying to connect before failing.
* `nats-mq/core/test_util.go` has the implementation used to run the nats server, the streaming server and the MQ image for each test.
* A number of performance "tests" are provided in the `performance` folder.
* The tests start and stop the MQ docker image repeatedly so can take over 5 minutes to run, you can use `docker ps` to keep an eye on things, images shouldn't last more than a couple minutes