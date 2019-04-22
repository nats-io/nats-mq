FROM golang:1.12.4

LABEL maintainer "Stephen Asbury <sasbury@nats.io>"

LABEL "ProductName"="NATS-MQ Bridge" \
      "ProductVersion"="0.5"

# Install the MQ client from the Redistributable package. This also
# contains the header files we need to compile against.
RUN mkdir -p /opt/mqm && cd /opt/mqm \
 && curl -LO "https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.1.2.0-IBM-MQC-Redist-LinuxX64.tar.gz" \
 && tar -zxf ./*.tar.gz \
 && rm -f ./*.tar.gz

ENV CGO_CFLAGS="-I/opt/mqm/inc/"
ENV CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"

# Build the go ibmmq library
RUN go get github.com/ibm-messaging/mq-golang/ibmmq
RUN chmod -R a+rx $GOPATH/src
RUN cd $GOPATH/src \
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
