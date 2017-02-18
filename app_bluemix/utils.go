package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/crypto"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/util"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/spf13/viper"
)

func initPeerClient() (err error) {
	config.SetupTestConfig(".")

	peerClientConn, err = peer.NewPeerClientConnection()
	if err != nil {
		fmt.Printf("error connection to server at host:port = %s\n", viper.GetString("peer.address"))
		return
	}
	serverClient = pb.NewPeerClient(peerClientConn)

	return
}

func initCryptoClient(enrollID, enrollPWD string) (crypto.Client, error) {
	// RegisterClient
	if enrollPWD != "" {
		if err := crypto.RegisterClient(enrollID, nil, enrollID, enrollPWD); err != nil {
			return nil, err
		}
	}

	client, err := crypto.InitClient(enrollID, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func processTransaction(tx *pb.Transaction) (*pb.Response, error) {
	resp, err := serverClient.ProcessTransaction(context.Background(), tx)

	if err != nil {
		for i := 0; i < retryCount; i++ {
			err = initPeerClient()
			if err != nil {
				logger.Errorf("processTransaction[%v]: %v", i, err)
				continue
			}
			resp, err = serverClient.ProcessTransaction(context.Background(), tx)
			if err == nil {
				logger.Errorf("processTransaction[%v]: successful", i)
				return resp, err
			}
		}
	}

	return resp, err
}

func confidentiality(enabled bool) {
	confidentialityOn = enabled

	if confidentialityOn {
		confidentialityLevel = pb.ConfidentialityLevel_CONFIDENTIAL
	} else {
		confidentialityLevel = pb.ConfidentialityLevel_PUBLIC
	}
}

func getChaincodeBytes(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	mode := viper.GetString("chaincode.mode")
	var codePackageBytes []byte
	if mode != chaincode.DevModeUserRunsChaincode {
		logger.Debugf("Received build request for chaincode spec: %v", spec)
		var err error
		if err = checkSpec(spec); err != nil {
			return nil, err
		}

		codePackageBytes, err = container.GetChaincodePackageBytes(spec)
		if err != nil {
			err = fmt.Errorf("Error getting chaincode package bytes: %s", err)
			logger.Errorf("%s", err)
			return nil, err
		}
	}
	chaincodeDeploymentSpec := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackageBytes}
	return chaincodeDeploymentSpec, nil
}

func checkSpec(spec *pb.ChaincodeSpec) error {
	// Don't allow nil value
	if spec == nil {
		return errors.New("Expected chaincode specification, nil received")
	}

	platform, err := platforms.Find(spec.Type)
	if err != nil {
		return fmt.Errorf("Failed to determine platform type: %s", err)
	}

	return platform.ValidateSpec(spec)
}

func deployChaincode(string) error {
	// Get chaincode path
	chaincodePath := os.Getenv("CORE_APP_BLUE_CHAINCODEPATH")
	if chaincodePath == "" {
		chaincodePath = viper.GetString("app.blue.chaincodePath")
		if chaincodePath == "" {
			chaincodePath = "github.com/wutongtree/blue/chaincode"
		}
	}

	// Get deployer
	deployerID := os.Getenv("CORE_APP_BLUE_DEPLOYER")
	if deployerID == "" {
		deployerID = viper.GetString("app.blue.deployerID")
		if deployerID == "" {
			deployerID = "lukas"
		}
	}

	deployerSecret := os.Getenv("CORE_APP_BLUE_DEPLOYERSECRET")
	if deployerSecret == "" {
		deployerSecret = viper.GetString("app.blue.deployerSecret")
		if deployerSecret == "" {
			deployerSecret = "NPKYL39uKbkj"
		}
	}

	// init deployer
	client, err := initCryptoClient(deployerID, deployerSecret)
	if err != nil {
		logger.Debugf("Failed deploying [%s]", err)
		return err
	}
	deployerClient = client

	// Prepare the spec. The metadata includes the identity of the administrator
	// spec := &pb.ChaincodeSpec{
	// 	Type:                 1,
	// 	ChaincodeID:          &pb.ChaincodeID{Path: chaincodePath},
	// 	CtorMsg:              &pb.ChaincodeInput{Function: "init"},
	// 	ConfidentialityLevel: confidentialityLevel,
	// }

	spec := &pb.ChaincodeSpec{
		Type:                 1,
		ChaincodeID:          &pb.ChaincodeID{Path: chaincodePath},
		CtorMsg:              &pb.ChaincodeInput{Args: util.ToChaincodeArgs("init")},
		ConfidentialityLevel: confidentialityLevel,
	}

	// First build the deployment spec
	cds, err := getChaincodeBytes(spec)
	if err != nil {
		return fmt.Errorf("Error getting deployment spec: %s ", err)
	}

	logger.Infof("deployChaincode: %v", cds.ChaincodeSpec)

	// Now create the Transactions message and send to Peer.
	transaction, err := client.NewChaincodeDeployTransaction(cds, cds.ChaincodeSpec.ChaincodeID.Name)
	if err != nil {
		return fmt.Errorf("Error deploy chaincode: %s ", err)
	}

	resp, err := processTransaction(transaction)

	logger.Debugf("resp [%s]", resp.String())

	chaincodeName = cds.ChaincodeSpec.ChaincodeID.Name
	logger.Debugf("ChaincodeName [%s]", chaincodeName)

	return nil
}

func invokeChaincode(invoker crypto.Client, chaincodeInput *pb.ChaincodeInput) (resp *pb.Response, err error) {
	// Get a transaction handler to be used to submit the execute transaction
	txCertHandler, err := invoker.GetTCertificateHandlerNext()
	if err != nil {
		return nil, err
	}
	txHandler, err := txCertHandler.GetTransactionHandler()
	if err != nil {
		return nil, err
	}

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:                 1,
		ChaincodeID:          &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:              chaincodeInput,
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := txHandler.NewChaincodeExecute(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return nil, fmt.Errorf("Error invoke chaincode: %s ", err)
	}

	return processTransaction(transaction)
}

func queryChaincode(invoker crypto.Client, chaincodeInput *pb.ChaincodeInput) (resp *pb.Response, err error) {
	// Get a transaction handler to be used to submit the execute transaction
	txCertHandler, err := invoker.GetTCertificateHandlerNext()
	if err != nil {
		return nil, err
	}
	txHandler, err := txCertHandler.GetTransactionHandler()
	if err != nil {
		return nil, err
	}

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:                 1,
		ChaincodeID:          &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:              chaincodeInput,
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := txHandler.NewChaincodeQuery(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return nil, fmt.Errorf("Error query chaincode: %s ", err)
	}

	return processTransaction(transaction)
}

func getHTTPURL(resource string) string {
	var restServer = os.Getenv("CORE_REST_ADDRESS")
	if restServer == "" {
		restServer = viper.GetString("rest.address")
	}

	server := strings.Split(restServer, ":")
	if len(server) < 2 {
		return fmt.Sprintf("http://%v/%v", restServer, resource)
	}

	if server[1] == "443" {
		return fmt.Sprintf("https://%v/%v", server[0], resource)
	}

	return fmt.Sprintf("http://%v/%v", restServer, resource)
}

func serializeObject(obj interface{}) (string, error) {
	r, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	result := string(r)
	if result == "null" {
		return "", errors.New("null object")
	}

	return result, nil
}

func deserializeObject(str string) (interface{}, error) {
	var obj interface{}

	err := json.Unmarshal([]byte(str), &obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, errors.New("null object")
	}

	return obj, nil
}

func performHTTPGet(url string) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*3)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 60))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 60,
		},
	}
	rsp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func performHTTPPost(url string, b []byte) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*3)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 60))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 60,
		},
	}

	body := bytes.NewBuffer([]byte(b))
	res, err := client.Post(url, "application/json;charset=utf-8", body)
	if err != nil {

		return nil, err
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {

		return nil, err
	}

	return result, nil
}

func performHTTPDelete(url string) []byte {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil
	}

	return body
}
