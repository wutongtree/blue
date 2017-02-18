package main

import (
	"encoding/json"
	"fmt"
	"time"

	"net/http"

	"github.com/spf13/cobra"

	"github.com/gocraft/web"
	// "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/util"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/spf13/viper"
)

// --------------- AppCmd ---------------

// AppCmd returns the cobra command for APP
func AppCmd() *cobra.Command {
	return appStartCmd
}

var appStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the app.",
	Long:  `Starts a app that interacts with the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve(args)
	},
}

// --------------- Structs ---------------
// following defines structs used for communicate with blue

type BlueResponse struct {
	Status string `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
}

// --------------- BlueAPP ---------------

// BlueAPP defines the Blue REST service object.
type BlueAPP struct {
}

func buildBlueRouter() *web.Router {
	router := web.New(BlueAPP{})

	// Add middleware
	router.Middleware((*BlueAPP).SetResponseType)

	// Add routes
	router.Post("/tx/send", (*BlueAPP).Send)
	router.Post("/tx/offer", (*BlueAPP).Offer)

	// Add not found page
	router.NotFound((*BlueAPP).NotFound)

	return router
}

// SetResponseType is a middleware function that sets the appropriate response
// headers. Currently, it is setting the "Content-Type" to "application/json" as
// well as the necessary headers in order to enable CORS for Swagger usage.
func (s *BlueAPP) SetResponseType(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	rw.Header().Set("Content-Type", "application/json")

	// Enable CORS
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "accept, content-type")

	next(rw, req)
}

// NotFound returns a custom landing page when a given hyperledger end point
// had not been defined.
func (s *BlueAPP) NotFound(rw web.ResponseWriter, r *web.Request) {
	rw.WriteHeader(http.StatusNotFound)
	json.NewEncoder(rw).Encode(BlueResponse{Status: "Blue endpoint not found."})
}

// send send transactions
func (s *BlueAPP) Send(rw web.ResponseWriter, req *web.Request) {
	encoder := json.NewEncoder(rw)

	// get params
	sender := req.FormValue("sender")
	receiver := req.FormValue("receiver")
	amount := req.FormValue("amount")
	currency := req.FormValue("currency")

	logger.Infof("send: sender=%v receiver=%v amount=%v currency=%v", sender, receiver, amount, currency)

	// Check that the enrollId and enrollSecret are not left blank.
	if (sender == "") || (receiver == "") || (amount == "") || (currency == "") {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: "params error"})
		logger.Error("Error: params error.")

		return
	}

	// construct chaincodeInput
	location, _ := time.LoadLocation("Asia/Chongqing")
	timestr := time.Now().In(location).String()

	args := []string{
		"send",
		sender,
		receiver,
		amount,
		currency,
		timestr}

	// chaincodeInput := &pb.ChaincodeInput{
	// 	Function: "send",
	// 	Args:     args,
	// }
	chaincodeInput := &pb.ChaincodeInput{
		Args: util.ToChaincodeArgs(args...),
	}

	// invoke chaincode
	resp, err := invokeChaincode(deployerClient, chaincodeInput)
	if err != nil {
		errstr := fmt.Sprintf("offer error: %v", err)
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: errstr})
		logger.Error(errstr)

		return
	}
	if resp.Status != 200 {
		errstr := fmt.Sprintf("offer error: %v", err)
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: errstr})
		logger.Error(errstr)

		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(BlueResponse{Status: "success"})
	logger.Infof("send successful.\n")

	return
}

// offer offer transactions
func (s *BlueAPP) Offer(rw web.ResponseWriter, req *web.Request) {
	encoder := json.NewEncoder(rw)

	// get params
	sender := req.FormValue("sender")
	takerGets := req.FormValue("takerGets")
	takerPays := req.FormValue("takerPays")

	logger.Infof("offer: sender=%v takerGets=%v takerPays=%v", sender, takerGets, takerPays)

	// Check that the enrollId and enrollSecret are not left blank.
	if (sender == "") || (takerGets == "") || (takerPays == "") {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: "params error"})
		logger.Error("Error: params error.")

		return
	}

	// construct chaincodeInput
	location, _ := time.LoadLocation("Asia/Chongqing")
	timestr := time.Now().In(location).String()

	args := []string{
		"offer",
		sender,
		takerGets,
		takerPays,
		timestr}

	// chaincodeInput := &pb.ChaincodeInput{
	// 	Function: "offer",
	// 	Args:     args,
	// }
	chaincodeInput := &pb.ChaincodeInput{
		Args: util.ToChaincodeArgs(args...),
	}

	// invoke chaincode
	resp, err := invokeChaincode(deployerClient, chaincodeInput)
	if err != nil {
		errstr := fmt.Sprintf("offer error: %v", err)
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: errstr})
		logger.Error(errstr)

		return
	}

	if resp.Status != 200 {
		errstr := fmt.Sprintf("offer error: %v", err)
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(BlueResponse{Status: errstr})
		logger.Error(errstr)

		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(BlueResponse{Status: "success"})
	logger.Infof("offer successful: '%s'\n", resp.Msg)

	return
}

// --------------- common function --------------

// StartBlueServer initializes the REST service and adds the required
// middleware and routes.
func startBlueServer() {
	// Initialize the REST service object
	tlsEnabled := viper.GetBool("app.tls.enabled")

	logger.Infof("Initializing the REST service on %s, TLS is %s.", viper.GetString("app.address"), (map[bool]string{true: "enabled", false: "disabled"})[tlsEnabled])

	router := buildBlueRouter()

	// Start server
	if tlsEnabled {
		err := http.ListenAndServeTLS(viper.GetString("app.address"), viper.GetString("app.tls.cert.file"), viper.GetString("app.tls.key.file"), router)
		if err != nil {
			logger.Errorf("ListenAndServeTLS: %s", err)
		}
	} else {
		err := http.ListenAndServe(viper.GetString("app.address"), router)
		if err != nil {
			logger.Errorf("ListenAndServe: %s", err)
		}
	}
}

// start serve
func serve(args []string) error {
	// Create and register the REST service if configured
	startBlueServer()

	logger.Infof("Starting app...")

	return nil
}
