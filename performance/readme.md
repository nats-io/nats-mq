# Performance Tests/Examples/Metrics

This folder contains several apps to look at performance of the bridge:

* `encodingperf` - measures performance encoding and decoding messages.
* `full` - runs a set of messages through the bridge from queue -> NATS, with an external bridge and mq server. Messages are 1024 bytes long.
* `full_testenv` - runs a set of messages through MQ -> NATs and measures the total time. The test environment from test_utils.go is used for the bridge, nats and MQ server. Messages are 1024 bytes long.
* `multiqueue_testenv` - prepares multiple queues with messages and then runs the bridge reading those messages to nats.  The test environment from test_utils.go is used for the bridge, nats and MQ server. Has an option to run with TLS. Messages are 1024 bytes long.
* `queues` - runs a set of messages through MQ from queue -> queue, with an external mq server. Messages are 1024 bytes long. This test is useful for comparing performance to `full`.
* `singlequeue_testenv` - prepares a single queue with messages and then runs the bridge reading those messages to nats.  The test environment from test_utils.go is used for the bridge, nats and MQ server. Messages are 1024 bytes long.