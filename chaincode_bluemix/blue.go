package main

import (
	"errors"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/crypto/primitives"
	"github.com/op/go-logging"
)

// For environment variables.
var (
	logger = logging.MustGetLogger("blue.chaincode")

	sHandler = NewBlueHandler()
)

// restResult defines the response payload for a general REST interface request.
type restResult struct {
	OK    string `protobuf:"bytes,1,opt,name=OK" json:"OK,omitempty"`
	Error string `protobuf:"bytes,2,opt,name=Error" json:"Error,omitempty"`
}

//BlueChaincode APIs exposed to chaincode callers
type BlueChaincode struct {
}

// send send transactions
// args[0]: sender
// args[1]: receiver
// args[2]: amount
// args[3]: currency
// args[4]: timestr
func (t *BlueChaincode) send(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	logger.Debugf("+++++++++++++++++++++++++++++++++++ send in chaincode +++++++++++++++++++++++++++++++++")
	logger.Debugf("send args: %v", args)

	// parse arguments
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	sender := args[0]
	receiver := args[1]
	amount := args[2]
	currency := args[3]
	timestr := args[4]

	// save state
	return nil, sHandler.submitSend(stub,
		timestr,
		sender,
		receiver,
		amount,
		currency)
}

// offer offer transactions
// args[0]: sender
// args[1]: takerGets
// args[2]: takerPays
// args[3]: timestr
func (t *BlueChaincode) offer(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	logger.Debugf("+++++++++++++++++++++++++++++++++++ send in chaincode +++++++++++++++++++++++++++++++++")
	logger.Debugf("send args: %v", args)

	// parse arguments
	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	sender := args[0]
	takerGets := args[1]
	takerPays := args[2]
	timestr := args[3]

	// save state
	return nil, sHandler.submitOffer(stub,
		timestr,
		sender,
		takerGets,
		takerPays)
}

// ----------------------- CHAINCODE ----------------------- //

// Init initialization, this method will create asset despository in the chaincode state
func (t *BlueChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	logger.Debugf("********************************Init****************************************")

	logger.Info("[BlueChaincode] Init")
	if len(args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	return nil, sHandler.createTable(stub)
}

// Invoke  method is the interceptor of all invocation transactions, its job is to direct
// invocation transactions to intended APIs
func (t *BlueChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	logger.Debugf("********************************Invoke****************************************")

	//	 Handle different functions
	if function == "send" {
		// Sign file
		return t.send(stub, args)
	} else if function == "offer" {
		// Verify file
		return t.offer(stub, args)
	}

	return nil, errors.New("Received unknown function invocation")
}

// Query method is the interceptor of all invocation transactions, its job is to direct
// query transactions to intended APIs, and return the result back to callers
func (t *BlueChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	logger.Debugf("********************************Query****************************************")

	return nil, errors.New("Received unknown function query invocation with function " + function)
}

func main() {
	// chaincode won't read the yaml, so set the security leverl mannually
	primitives.SetSecurityLevel("SHA3", 256)
	err := shim.Start(new(BlueChaincode))
	if err != nil {
		logger.Debugf("Error starting BlueChaincode: %s", err)
	}
}
