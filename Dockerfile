FROM ubuntu:18.10

LABEL maintainer "Stephen Asbury <sasbury@nats.io>"

LABEL "ProductName"="NATS-MQ Bridge" \
      "ProductVersion"="0.5"

ENV GOPATH=/go

# Install the Go compiler and Git
RUN export DEBIAN_FRONTEND=noninteractive \
  && bash -c 'source /etc/os-release; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME} main restricted" > /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-updates main restricted" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-backports main restricted universe" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME} universe" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-updates universe" >> /etc/apt/sources.list;' \
  && apt-get update \
  && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    curl \
    tar \
    bash \
    build-essential \
    software-properties-common \
  && add-apt-repository ppa:gophers/archive \
  && apt-get update \
  && apt-get install -y --no-install-recommends \
    golang-1.11-go \
    go-dep \
  && rm -rf /var/lib/apt/lists/*

# Create location for the git clone and MQ installation
RUN mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg \
  && chmod -R 777 $GOPATH \
  && mkdir -p $GOPATH/src/$ORG \
  && mkdir -p /opt/mqm \
  && chmod a+rx /opt/mqm

# Install the MQ client from the Redistributable package. This also contains the
# header files we need to compile against.
RUN cd /opt/mqm \
 && curl -LO "https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.1.2.0-IBM-MQC-Redist-LinuxX64.tar.gz" \
 && tar -zxf ./*.tar.gz \
 && rm -f ./*.tar.gz

ENV PATH="${PATH}:/usr/lib/go-1.11/bin:/go/bin"
ENV GOROOT="/usr/lib/go-1.11"
ENV CGO_CFLAGS="-I/opt/mqm/inc/"
ENV CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
ENV GOCACHE=/tmp/.gocache

# Build the go ibmmq library
RUN go get github.com/ibm-messaging/mq-golang/ibmmq
RUN chmod -R a+rx $GOPATH/src
RUN cd ${GOPATH}/src \
  && go install github.com/ibm-messaging/mq-golang/ibmmq

# Copy and build the nats-mq code
RUN mkdir -p /nats-mq \
  && chmod -R 777 /nats-mq
COPY . /nats-mq
RUN rm -rf /nats-mq/build /nats-mq/.vscode
RUN chmod -R a+rx /nats-mq

RUN cd /nats-mq && go mod download && go install ./...

# Run the bridge
ENTRYPOINT ["/go/bin/nats-mq","-c","/mqbridge.conf"]
