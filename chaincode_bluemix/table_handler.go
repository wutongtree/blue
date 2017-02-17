package main

import (
	"errors"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// consts associated with chaincode table
const (
	// table
	tableSend  = "send"
	tableOffer = "offer"

	// column
	columnSender    = "sender"
	columnReceiver  = "receiver"
	columnAmount    = "amount"
	columnCurrency  = "currency"
	columnTakerGets = "takerGets"
	columnTakerPays = "takerPays"
	columnTimestamp = "timestamp"
)

//BlueHandler provides APIs used to perform operations on CC's KV store
type tableHandler struct {
}

// NewBlueHandler create a new reference to CertHandler
func NewBlueHandler() *tableHandler {
	return &tableHandler{}
}

// createTable
// stub: chaincodestub
func (t *tableHandler) createTable(stub shim.ChaincodeStubInterface) error {

	// Create send table
	err := stub.CreateTable(tableSend, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: columnTimestamp, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnSender, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnReceiver, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnAmount, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnCurrency, Type: shim.ColumnDefinition_STRING, Key: false},
	})

	if err != nil {
		logger.Errorf("createTable error: %v", err)
		return err
	}

	// Create offer table
	err = stub.CreateTable(tableOffer, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: columnTimestamp, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnSender, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnTakerGets, Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: columnTakerPays, Type: shim.ColumnDefinition_STRING, Key: true},
	})

	if err != nil {
		logger.Errorf("createTable error: %v", err)
	}

	return err
}

// submitSend submit send
// sender: sender
// receiver: receiver
// amount: amount
// currency: currency
// timestamp: timestamp
func (t *tableHandler) submitSend(stub shim.ChaincodeStubInterface,
	sender string,
	receiver string,
	amount string,
	currency string,
	timestamp string) error {

	logger.Debugf("insert table send: sender=%v receiver=%v amount=%v currency=%v timestamp=%v", sender, receiver, amount, currency, timestamp)

	//insert a new row for send transaction
	ok, err := stub.InsertRow(tableSend, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: timestamp}},
			&shim.Column{Value: &shim.Column_String_{String_: sender}},
			&shim.Column{Value: &shim.Column_String_{String_: receiver}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: currency}}},
	})

	// you can only assign balances to new account IDs
	if !ok && err == nil {
		logger.Errorf("submitSend: system error %v", err)
		return errors.New("Fiel was already signed.")
	}

	return nil
}

// submitSend submit offer
// sender: sender
// takerGets: takerGets
// takerPays: takerPays
// timestamp: timestamp
func (t *tableHandler) submitOffer(stub shim.ChaincodeStubInterface,
	sender string,
	takerGets string,
	takerPays string,
	timestamp string) error {

	logger.Debugf("insert table send: sender=%v takerGets=%v takerPays=%v timestamp=%v", sender, takerGets, takerPays, timestamp)

	//insert a new row for send transaction
	ok, err := stub.InsertRow(tableSend, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: timestamp}},
			&shim.Column{Value: &shim.Column_String_{String_: sender}},
			&shim.Column{Value: &shim.Column_String_{String_: takerGets}},
			&shim.Column{Value: &shim.Column_String_{String_: takerPays}}},
	})

	// you can only assign balances to new account IDs
	if !ok && err == nil {
		logger.Errorf("submitSend: system error %v", err)
		return errors.New("Fiel was already signed.")
	}

	return nil
}
