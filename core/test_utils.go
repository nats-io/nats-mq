package core

import (
	"encoding/json"
	"fmt"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nuid"
	"log"
	"os/exec"
	"strings"
	"time"

	gnatsserver "github.com/nats-io/gnatsd/server"
	gnatsd "github.com/nats-io/gnatsd/test"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	nss "github.com/nats-io/nats-streaming-server/server"
)

// TestEnv encapsulate a bridge test environment
type TestEnv struct {
	MQServer *MQTestServer
	QMgr     *ibmmq.MQQueueManager // For bypassing the bridge connection

	Gnatsd *gnatsserver.Server
	Stan   *nss.StanServer

	NC *nats.Conn // for bypassing the bridge
	SC stan.Conn  // for bypassing the bridge

	Bridge *BridgeServer
	Config *BridgeConfig
}

// StartTestEnvironment sreates a new test environment and bridge server
// The MQ setting on the connectionconfigs will be filled in
func StartTestEnvironment(connections []ConnectionConfig) (*TestEnv, error) {
	tbs := &TestEnv{}

	mqServer, qmgr, err := StartMQTestServer(5 * time.Second)
	if err != nil {
		return nil, err
	}
	tbs.MQServer = mqServer
	tbs.QMgr = qmgr

	opts := gnatsd.DefaultTestOptions
	opts.Port = -1
	tbs.Gnatsd = gnatsd.RunServer(&opts)

	natsServerURL := fmt.Sprintf("nats://localhost:%d", opts.Port)

	clusterName := nuid.Next()
	sOpts := nss.GetDefaultOptions()
	sOpts.ID = clusterName
	sOpts.NATSServerURL = natsServerURL
	nOpts := nss.DefaultNatsServerOptions
	nOpts.Port = -1

	s, err := nss.RunServerWithOpts(sOpts, &nOpts)
	if err != nil {
		tbs.Close()
		return nil, err
	}

	tbs.Stan = s

	nc, err := nats.Connect(natsServerURL)
	if err != nil {
		tbs.Close()
		return nil, err
	}
	tbs.NC = nc

	sc, err := stan.Connect(clusterName, nuid.Next(), stan.NatsConn(tbs.NC))
	if err != nil {
		tbs.Close()
		return nil, err
	}
	tbs.SC = sc

	config := DefaultBridgeConfig()
	config.NATS = NATSConfig{
		Servers:        []string{natsServerURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}
	config.STAN = NATSStreamingConfig{
		ClusterID:          clusterName,
		ClientID:           nuid.Next(),
		PubAckWait:         5000,
		DiscoverPrefix:     stan.DefaultDiscoverPrefix,
		MaxPubAcksInflight: stan.DefaultMaxPubAcksInflight,
		ConnectWait:        2000,
	}

	for i, c := range connections {
		c.MQ = MQConfig{
			ConnectionName: mqServer.AppHostPort,
			ChannelName:    "DEV.APP.SVRCONN",
			QueueManager:   "QM1",
		}
		connections[i] = c
	}

	config.Connect = connections

	tbs.Config = &config
	tbs.Bridge = NewBridgeServer()
	err = tbs.Bridge.LoadConfig(config)
	if err != nil {
		tbs.Close()
		return nil, err
	}
	err = tbs.Bridge.Start()
	if err != nil {
		tbs.Close()
		return nil, err
	}

	return tbs, nil
}

func (tbs *TestEnv) getQueueManagerName() string {
	return "QM1"
}

// Use the test environments extra connection to talk to the queue, bypassing the bridge's connection
func (tbs *TestEnv) getMessageFromQueue(qName string, waitMillis int32) (*ibmmq.MQMD, []byte, error) {
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = qName

	qObject, err := tbs.QMgr.Open(mqod, openOptions)
	if err != nil {
		return nil, nil, err
	}
	defer qObject.Close(0)

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()

	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = waitMillis

	buffer := make([]byte, 4096)
	datalen, err := qObject.Get(getmqmd, gmo, buffer)

	if err != nil {
		return nil, nil, err
	}

	return getmqmd, buffer[:datalen], nil
}

// Use the test environments extra connection to talk to the queue, bypassing the bridge's connection
func (tbs *TestEnv) putMessageOnQueue(qName string, mqmd *ibmmq.MQMD, msgData []byte) error {
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = qName // Note queue uses name, topic uses string
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	buffer := []byte(msgData)

	return tbs.QMgr.Put1(mqod, mqmd, pmo, buffer)
}

// Use the test environments extra connection to talk to the topic, bypassing the bridge's connection
func (tbs *TestEnv) putMessageOnTopic(topicName string, mqmd *ibmmq.MQMD, msgData []byte) error {
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_TOPIC
	mqod.ObjectString = topicName // Note queue uses name, topic uses string
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	buffer := []byte(msgData)

	return tbs.QMgr.Put1(mqod, mqmd, pmo, buffer)
}

// Close the bridge server and clean up the test environment
func (tbs *TestEnv) Close() {
	// Stop the bridge first!
	if tbs.Bridge != nil {
		tbs.Bridge.Stop()
	}

	if tbs.QMgr != nil {
		tbs.QMgr.Disc()
	}

	if tbs.MQServer != nil {
		tbs.MQServer.Close()
	}

	if tbs.SC != nil {
		tbs.SC.Close()
	}

	if tbs.NC != nil {
		tbs.NC.Close()
	}

	if tbs.Stan != nil {
		tbs.Stan.Shutdown()
	}
}

// MQTestServer is based on - https://ericchiang.github.io/post/testing-dbs-with-docker/
// MQTestServer wraps an MQ server running in docker
type MQTestServer struct {
	QueueManager string
	CID          string
	AppHostPort  string
	WebHostPort  string
}

//StartMQTestServer creates a test db in docker
func StartMQTestServer(waitForStart time.Duration) (*MQTestServer, *ibmmq.MQQueueManager, error) {
	start := time.Now()
	img := "ibmcom/mq"

	if exec.Command("docker", "inspect", img).Run() != nil {
		return nil, nil, fmt.Errorf("db requires docker image %s, please pull or specify a different version", img)
	}

	// Running on port 0 instructs the operating system to pick an available port.
	dockerArgs := []string{"run", "--publish", "0:1414", "--publish", "0:9443", "--detach"}
	envvars := map[string]string{
		"LICENSE":      "accept",
		"MQ_QMGR_NAME": "QM1",
	}
	for key, val := range envvars {
		if val != "" {
			dockerArgs = append(dockerArgs, "--env", key+"="+val)
		}
	}
	dockerArgs = append(dockerArgs, img)

	out, err := exec.Command("docker", dockerArgs...).CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("docker run: %v: %s", err, out)
	}
	cid := strings.TrimSpace(string(out))
	mq := &MQTestServer{CID: cid, QueueManager: "QM1"}

	_, port, err := mq.portMapping("1414/tcp")
	if err != nil {
		mq.Close()
		return nil, nil, err
	}
	mq.AppHostPort = fmt.Sprintf("localhost(%s)", port)

	_, port, err = mq.portMapping("9443/tcp")
	if err != nil {
		mq.Close()
		return nil, nil, err
	}
	mq.WebHostPort = fmt.Sprintf("localhost:%s", port)

	config := MQConfig{
		ConnectionName: mq.AppHostPort,
		ChannelName:    "DEV.APP.SVRCONN",
		QueueManager:   "QM1",
	}

	var connection *ibmmq.MQQueueManager

	for waitForStart > 0 && time.Now().Sub(start) < waitForStart {
		connection, err = ConnectToQueueManager(config)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("started MQ series docker image at %s\n", mq.CID[0:12])
	return mq, connection, nil
}

// Close a test db
func (mq *MQTestServer) Close() error {
	log.Printf("stopping MQ series docker image at %s\n", mq.CID[0:12])
	out, err := exec.Command("docker", "rm", "-f", mq.CID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm: %v: %s", err, out)
	}
	return nil
}

func (mq *MQTestServer) portMapping(containerPort string) (hostAddr string, hostPort string, err error) {
	out, err := exec.Command("docker", "inspect", mq.CID).CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("docker inspect: %v: %s", err, out)
	}

	// anonymous struct for unmarshalling JSON into
	var inspectResp []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIP   string
				HostPort string
			}
		}
	}
	if err := json.Unmarshal(out, &inspectResp); err != nil {
		return "", "", fmt.Errorf("decoding docker inspect result failed: %v: %s", err, out)
	}
	if len(inspectResp) != 1 {
		return "", "", fmt.Errorf("expected one inspect result, got %d", len(inspectResp))
	}
	ports := inspectResp[0].NetworkSettings.Ports[containerPort]
	if len(ports) != 1 {
		return "", "", fmt.Errorf("expected one port mapping, got %d", len(ports))
	}
	return ports[0].HostIP + "(" + ports[0].HostPort + ")", ports[0].HostPort, nil
}
