#!/bin/sh

docker run \
  --volume `pwd`/resources/mqm:/mnt/tls \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_APP_PASSWORD=passw0rd \
  --env MQ_TLS_KEYSTORE=/mnt/tls/MQServer/certs/key.p12 \
  --env MQ_TLS_PASSPHRASE=k3ypassw0rd \
  --publish 1414:1414 \
  --publish 9443:9443 \
  ibmcom/mq
