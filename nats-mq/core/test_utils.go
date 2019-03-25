package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	gnatsserver "github.com/nats-io/gnatsd/server"
	gnatsd "github.com/nats-io/gnatsd/test"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	nss "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nuid"
)

// TestEnv encapsulate a bridge test environment
type TestEnv struct {
	MQServer *MQTestServer
	QMgr     *ibmmq.MQQueueManager // For bypassing the bridge connection

	Gnatsd *gnatsserver.Server
	Stan   *nss.StanServer

	NC *nats.Conn // for bypassing the bridge
	SC stan.Conn  // for bypassing the bridge

	natsPort       int
	natsURL        string
	clusterName    string
	clientID       string // we keep this so we stay the same on reconnect
	bridgeClientID string

	Bridge *BridgeServer
	Config *conf.BridgeConfig
}

// StartTestEnvironment calls StartTestEnvironmentInfrastructure
// followed by StartBridge
func StartTestEnvironment(connections []conf.ConnectorConfig) (*TestEnv, error) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	if err != nil {
		return nil, err
	}
	err = tbs.StartBridge(connections, false)
	if err != nil {
		tbs.Close()
		return nil, err
	}
	return tbs, err
}

// StartTLSTestEnvironment calls StartTestEnvironmentInfrastructure
// followed by StartBridge, with TLS enabled
func StartTLSTestEnvironment(connections []conf.ConnectorConfig) (*TestEnv, error) {
	tbs, err := StartTestEnvironmentInfrastructure(true)
	if err != nil {
		return nil, err
	}
	err = tbs.StartBridge(connections, true)
	if err != nil {
		tbs.Close()
		return nil, err
	}
	return tbs, err
}

// StartTestEnvironmentInfrastructure creates the MQMgr, Nats and streaming
// but does not start a bridge, you can use StartBridge to start a bridge afterward
func StartTestEnvironmentInfrastructure(useTLS bool) (*TestEnv, error) {
	tbs := &TestEnv{}

	mqServer, qmgr, err := StartMQTestServer(30*time.Second, useTLS, 0) // port 0 -> ephemeral
	if err != nil {
		return nil, err
	}
	tbs.MQServer = mqServer
	tbs.QMgr = qmgr

	err = tbs.StartNATSandStan(useTLS, -1, nuid.Next(), nuid.Next(), nuid.Next())
	if err != nil {
		tbs.Close()
		return nil, err
	}

	return tbs, nil
}

// StartBridge is the second half of StartTestEnvironment
// it is provided separately so that environment can be created before the bridge runs
func (tbs *TestEnv) StartBridge(connections []conf.ConnectorConfig, useTLS bool) error {
	config := conf.DefaultBridgeConfig()
	//config.Logging.Debug = true
	//config.Logging.Trace = true
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort: -1,
	}
	config.NATS = conf.NATSConfig{
		Servers:        []string{tbs.natsURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}
	config.STAN = conf.NATSStreamingConfig{
		ClusterID:          tbs.clusterName,
		ClientID:           tbs.bridgeClientID,
		PubAckWait:         5000,
		DiscoverPrefix:     stan.DefaultDiscoverPrefix,
		MaxPubAcksInflight: stan.DefaultMaxPubAcksInflight,
		ConnectWait:        2000,
	}

	if useTLS {
		config.Monitoring.HTTPPort = 0
		config.Monitoring.HTTPSPort = -1

		config.Monitoring.TLS = conf.TLSConf{
			Cert: "../../resources/certs/server-cert.pem",
			Key:  "../../resources/certs/server-key.pem",
		}

		config.NATS.TLS = conf.TLSConf{
			Root: "../../resources/certs/ca.pem",
		}
	}

	clientKeyFile, err := filepath.Abs("../../resources/mqm/MQClient/certs/client")

	if err != nil {
		return err
	}

	for i, c := range connections {
		c.MQ = conf.MQConfig{
			ConnectionName: tbs.MQServer.AppHostPort,
			ChannelName:    "DEV.APP.SVRCONN",
			QueueManager:   "QM1",
		}

		if useTLS {
			c.MQ.UserName = "app"
			c.MQ.Password = "passw0rd"
			c.MQ.KeyRepository = clientKeyFile
			c.MQ.CertificateLabel = "QM1.cert"
		}

		connections[i] = c
	}

	config.Connect = connections

	tbs.Config = &config
	tbs.Bridge = NewBridgeServer()
	err = tbs.Bridge.LoadConfig(config)
	if err != nil {
		tbs.Close()
		return err
	}
	err = tbs.Bridge.Start()
	if err != nil {
		tbs.Close()
		return err
	}

	return nil
}

// StartNATSandStan starts up the nats and stan servers
func (tbs *TestEnv) StartNATSandStan(useTLS bool, port int, clusterID string, clientID string, bridgeClientID string) error {
	var err error
	opts := gnatsd.DefaultTestOptions
	opts.Port = port

	if useTLS {
		opts.TLSCert = "../../resources/certs/server-cert.pem"
		opts.TLSKey = "../../resources/certs/server-key.pem"
		opts.TLSTimeout = 5

		tc := gnatsserver.TLSConfigOpts{}
		tc.CertFile = opts.TLSCert
		tc.KeyFile = opts.TLSKey

		opts.TLSConfig, err = gnatsserver.GenTLSConfig(&tc)

		if err != nil {
			return err
		}
	}
	tbs.Gnatsd = gnatsd.RunServer(&opts)

	if useTLS {
		tbs.natsURL = fmt.Sprintf("tls://localhost:%d", opts.Port)
	} else {
		tbs.natsURL = fmt.Sprintf("nats://localhost:%d", opts.Port)
	}

	tbs.natsPort = opts.Port
	tbs.clusterName = clusterID
	sOpts := nss.GetDefaultOptions()
	sOpts.ID = tbs.clusterName
	sOpts.NATSServerURL = tbs.natsURL

	if useTLS {
		sOpts.ClientCA = "../../resources/certs/ca.pem"
	}

	nOpts := nss.DefaultNatsServerOptions
	nOpts.Port = -1

	s, err := nss.RunServerWithOpts(sOpts, &nOpts)
	if err != nil {
		return err
	}

	tbs.Stan = s
	tbs.clientID = clientID
	tbs.bridgeClientID = bridgeClientID

	var nc *nats.Conn

	if useTLS {
		nc, err = nats.Connect(tbs.natsURL, nats.RootCAs("../../resources/certs/ca.pem"))
	} else {
		nc, err = nats.Connect(tbs.natsURL)
	}

	if err != nil {
		return err
	}

	tbs.NC = nc

	sc, err := stan.Connect(tbs.clusterName, tbs.clientID, stan.NatsConn(tbs.NC))
	if err != nil {
		return err
	}
	tbs.SC = sc

	return nil
}

// StopBridge stops the bridge
func (tbs *TestEnv) StopBridge() {
	if tbs.Bridge != nil {
		tbs.Bridge.Stop()
		tbs.Bridge = nil
	}
}

// RestartMQ shuts down the MQ server and then starts it again
func (tbs *TestEnv) RestartMQ(useTLS bool) error {
	if tbs.QMgr != nil {
		tbs.QMgr.Disc()
	}

	if tbs.MQServer != nil {
		tbs.MQServer.Close()
	}

	mqServer, qmgr, err := StartMQTestServer(30*time.Second, useTLS, tbs.MQServer.AppPort)
	if err != nil {
		return err
	}
	tbs.MQServer = mqServer
	tbs.QMgr = qmgr

	return nil
}

// StopNATS shuts down the NATS and Stan servers
func (tbs *TestEnv) StopNATS() error {
	if tbs.SC != nil {
		tbs.SC.Close()
	}

	if tbs.NC != nil {
		tbs.NC.Close()
	}

	if tbs.Stan != nil {
		tbs.Stan.Shutdown()
	}

	if tbs.Gnatsd != nil {
		tbs.Gnatsd.Shutdown()
	}

	return nil
}

// RestartNATS shuts down the NATS and stan server and then starts it again
func (tbs *TestEnv) RestartNATS(useTLS bool) error {
	if tbs.SC != nil {
		tbs.SC.Close()
	}

	if tbs.NC != nil {
		tbs.NC.Close()
	}

	if tbs.Stan != nil {
		tbs.Stan.Shutdown()
	}

	if tbs.Gnatsd != nil {
		tbs.Gnatsd.Shutdown()
	}

	err := tbs.StartNATSandStan(useTLS, tbs.natsPort, tbs.clusterName, tbs.clientID, tbs.bridgeClientID)
	if err != nil {
		return err
	}

	return nil
}

// GetQueueManagerName get the queue manager name for the test MQ server
func (tbs *TestEnv) GetQueueManagerName() string {
	return "QM1"
}

// GetMessageFromQueue uses the test environments extra connection to talk to the queue, bypassing the bridge's connection
func (tbs *TestEnv) GetMessageFromQueue(qName string, waitMillis int32) (*ibmmq.MQMD, []byte, error) {
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

// PutMessageOnQueue uses the test environments extra connection to talk to the queue, bypassing the bridge's connection
func (tbs *TestEnv) PutMessageOnQueue(qName string, mqmd *ibmmq.MQMD, msgData []byte) error {
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = qName // Note queue uses name, topic uses string
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	buffer := []byte(msgData)

	return tbs.QMgr.Put1(mqod, mqmd, pmo, buffer)
}

// PutMessageOnTopic uses the test environments extra connection to talk to the topic, bypassing the bridge's connection
func (tbs *TestEnv) PutMessageOnTopic(topicName string, mqmd *ibmmq.MQMD, msgData []byte) error {
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

	if tbs.Gnatsd != nil {
		tbs.Gnatsd.Shutdown()
	}
}

// MQTestServer is based on - https://ericchiang.github.io/post/testing-dbs-with-docker/
// MQTestServer wraps an MQ server running in docker
type MQTestServer struct {
	QueueManager string
	CID          string
	AppHostPort  string
	WebHostPort  string
	AppPort      int
}

//StartMQTestServer creates a test db in docker
func StartMQTestServer(waitForStart time.Duration, useTLS bool, mqPort int) (*MQTestServer, *ibmmq.MQQueueManager, error) {
	start := time.Now()
	img := "ibmcom/mq"

	if exec.Command("docker", "inspect", img).Run() != nil {
		return nil, nil, fmt.Errorf("db requires docker image %s, please pull or specify a different version", img)
	}

	// Running on port 0 instructs the operating system to pick an available port.
	dockerArgs := []string{"run", "--publish", fmt.Sprintf("%d:1414", mqPort), "--publish", "0:9443", "--detach"}
	envvars := map[string]string{
		"LICENSE":      "accept",
		"MQ_QMGR_NAME": "QM1",
	}

	dir, err := ioutil.TempDir("/tmp", "mqdata")
	extraQueues := ""

	// Add the queues
	testQueues := 25
	for i := 1; i <= testQueues; i++ {
		extraQueues += fmt.Sprintf("DEFINE QLOCAL(TEST.QUEUE.%d) REPLACE\n", i)
	}

	// set the permissions
	for i := 1; i <= testQueues; i++ {
		extraQueues += fmt.Sprintf("SET AUTHREC PROFILE('TEST.QUEUE.%d') OBJTYPE(QUEUE) PRINCIPAL('app') AUTHADD(BROWSE, GET, PUT, INQ)\n", i)
	}

	extraQueues += "ALTAR CHANNEL(DEV.APP.SVRCONN) SHARECNV(1)"

	extraQueueFile := filepath.Join(dir, "20-config.mqsc")
	ioutil.WriteFile(extraQueueFile, []byte(extraQueues), 0644)

	dockerArgs = append(dockerArgs, "--volume", fmt.Sprintf("%s:%s", extraQueueFile, "/etc/mqm/20-config.mqsc"))

	if useTLS {
		pwd, err := os.Getwd()

		if err != nil {
			return nil, nil, err
		}

		// Move up from nats-mq/core to root
		pwd = filepath.Dir(pwd)
		pwd = filepath.Dir(pwd)

		dockerArgs = append(dockerArgs, "--volume", fmt.Sprintf("%s:%s", filepath.Join(pwd, "/resources/mqm"), "/mnt/tls"))
		envvars["MQ_APP_PASSWORD"] = "passw0rd"
		envvars["MQ_TLS_KEYSTORE"] = "/mnt/tls/MQServer/certs/key.p12"
		envvars["MQ_TLS_PASSPHRASE"] = "k3ypassw0rd"
	}
	for key, val := range envvars {
		if val != "" {
			dockerArgs = append(dockerArgs, "--env", key+"="+val)
		}
	}
	dockerArgs = append(dockerArgs, img)

	fmt.Println(strings.Join(dockerArgs, " "))

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
	mq.AppPort, err = strconv.Atoi(port)
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

	config := conf.MQConfig{
		ConnectionName: mq.AppHostPort,
		ChannelName:    "DEV.APP.SVRCONN",
		QueueManager:   "QM1",
	}

	if useTLS {
		config.UserName = "app"
		config.Password = "passw0rd"

		fullPath, err := filepath.Abs("../../resources/mqm/MQClient/certs/client")

		if err != nil {
			return nil, nil, err
		}

		config.KeyRepository = fullPath
		config.CertificateLabel = "QM1.cert"
	}

	var connection *ibmmq.MQQueueManager

	for waitForStart > 0 && time.Now().Sub(start) < waitForStart {
		connection, err = ConnectToQueueManager(config)
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if connection == nil {
		mq.Close()
		return nil, nil, fmt.Errorf("unable to connect to MQ Manager")
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
