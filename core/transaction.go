package core

import (
	"ark-go/arkcoin"
	"ark-go/arkcoin/base58"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

//Transaction struct - represents structure of ARK.io blockchain transaction
type Transaction struct {
	Timestamp             int32  `json:"timestamp,omitempty"`
	RecipientID           string `json:"recipientId,omitempty"`
	Amount                int64  `json:"amount,omitempty"`
	Asset                 string `json:"asset,omitempty"`
	Fee                   int64  `json:"fee,omitempty"`
	Type                  byte   `json:"type"`
	VendorField           string `json:"vendorField,omitempty"`
	Signature             string `json:"signature,omitempty"`
	SignSignature         string `json:"signSignature,omitempty"`
	SenderPublicKey       string `json:"senderPublicKey,omitempty"`
	SecondSenderPublicKey string `json:"secondSenderPublicKey,omitempty"`
	RequesterPublicKey    string `json:"requesterPublicKey,omitempty"`
	ID                    string `json:"id,omitempty"`
}

//ToBytes returns bytearray of the Transaction object to be signed and send to blockchain
func (tx *Transaction) toBytes(skipSignature, skipSecondSignature bool) []byte {
	txBuf := new(bytes.Buffer)
	binary.Write(txBuf, binary.LittleEndian, tx.Type)
	binary.Write(txBuf, binary.LittleEndian, uint32(tx.Timestamp))

	binary.Write(txBuf, binary.LittleEndian, quickHexDecode(tx.SenderPublicKey))

	if tx.RequesterPublicKey != "" {
		res, err := base58.Decode(tx.RequesterPublicKey)
		if err != nil {
			binary.Write(txBuf, binary.LittleEndian, res)
		}
	}

	if tx.RecipientID != "" {
		res, err := base58.Decode(tx.RecipientID)
		if err != nil {
			log.Fatal("Error converting Decoding b58 ", err.Error())
		}
		binary.Write(txBuf, binary.LittleEndian, res)
	} else {
		binary.Write(txBuf, binary.LittleEndian, make([]byte, 21))
	}

	if tx.VendorField != "" {
		vendorBytes := []byte(tx.VendorField)
		if len(vendorBytes) < 65 {
			binary.Write(txBuf, binary.LittleEndian, vendorBytes)

			bs := make([]byte, 64-len(vendorBytes))
			binary.Write(txBuf, binary.LittleEndian, bs)
		}
	} else {
		binary.Write(txBuf, binary.LittleEndian, make([]byte, 64))
	}

	binary.Write(txBuf, binary.LittleEndian, uint64(tx.Amount))
	binary.Write(txBuf, binary.LittleEndian, uint64(tx.Fee))

	switch tx.Type {
	case 1:
		binary.Write(txBuf, binary.LittleEndian, quickHexDecode(tx.Signature))
	case 2:
		//TODO buffer.Put(Encoding.ASCII.GetBytes(asset["username"]));
	case 3:
		//TODO votes
	}

	if !skipSignature && len(tx.Signature) > 0 {
		binary.Write(txBuf, binary.LittleEndian, quickHexDecode(tx.Signature))
	}

	if !skipSecondSignature && len(tx.SignSignature) > 0 {
		binary.Write(txBuf, binary.LittleEndian, quickHexDecode(tx.SignSignature))
	}

	return txBuf.Bytes()
}

//CreateTransaction creates and returns new Transaction struct...
func CreateTransaction(recipientID string, satoshiAmount int64, vendorField, passphrase, secondPassphrase string) *Transaction {
	tx := Transaction{Type: 0,
		RecipientID: recipientID,
		Amount:      satoshiAmount,
		Fee:         arkcoin.ArkCoinMain.Fees.Send,
		VendorField: vendorField}

	tx.Timestamp = GetTime() //1
	tx.sign(passphrase)

	if len(secondPassphrase) > 0 {
		tx.secondSign(secondPassphrase)
	}

	tx.getID() //calculates id of transaction
	return &tx
}

//Sign the Transaction
func (tx *Transaction) sign(passphrase string) {
	key := arkcoin.NewPrivateKeyFromPassword(passphrase, arkcoin.ArkCoinMain)

	tx.SenderPublicKey = hex.EncodeToString(key.PublicKey.Serialize())

	trHashBytes := sha256.New()
	trHashBytes.Write(tx.toBytes(true, true))

	sig, err := key.Sign(trHashBytes.Sum(nil))
	if err == nil {
		tx.Signature = hex.EncodeToString(sig)
	}
}

//SecondSign the Transaction
func (tx *Transaction) secondSign(passphrase string) {
	key := arkcoin.NewPrivateKeyFromPassword(passphrase, arkcoin.ArkCoinMain)

	tx.SecondSenderPublicKey = hex.EncodeToString(key.PublicKey.Serialize())
	trHashBytes := sha256.New()
	trHashBytes.Write(tx.toBytes(false, true))

	sig, err := key.Sign(trHashBytes.Sum(nil))
	if err == nil {
		tx.SignSignature = hex.EncodeToString(sig)
	}
}

//GetID returns calculated ID of trancation - hashed s256
func (tx *Transaction) getID() {
	trHashBytes := sha256.New()
	trHashBytes.Write(tx.toBytes(false, false))

	tx.ID = hex.EncodeToString(trHashBytes.Sum(nil))
}

//ToJSON converts transaction object to JSON string
func (tx *Transaction) ToJSON() string {
	txJSON, err := json.Marshal(tx)
	if err != nil {
		log.Fatal(err.Error())
	}
	return string(txJSON)
}

func quickHexDecode(data string) []byte {
	res, err := hex.DecodeString(data)
	if err != nil {
		log.Fatal(err.Error())
	}
	return res
}

//Verify function verifies if tx is validly signed
//if return == nill verification was succesfull
func (tx *Transaction) Verify() error {
	key, err := arkcoin.NewPublicKey(quickHexDecode(tx.SenderPublicKey), arkcoin.ArkCoinMain)
	if err != nil {
		log.Fatal(err.Error())
	}
	trHashBytes := sha256.New()
	trHashBytes.Write(tx.toBytes(true, true))
	return key.Verify(quickHexDecode(tx.Signature), trHashBytes.Sum(nil))

}

//SecondVerify function verifies if tx is validly signed
//if return == nill verification was succesfull
func (tx *Transaction) SecondVerify() error {
	key, err := arkcoin.NewPublicKey(quickHexDecode(tx.SecondSenderPublicKey), arkcoin.ArkCoinMain)
	if err != nil {
		log.Fatal(err.Error())
	}
	trHashBytes := sha256.New()
	trHashBytes.Write(tx.toBytes(false, true))
	return key.Verify(quickHexDecode(tx.SignSignature), trHashBytes.Sum(nil))
}

//PostTransactionResponse structure for call /peer/list
type PostTransactionResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	Error          string   `json:"error"`
	TransactionIDs []string `json:"transactionIds"`
}

type transactionPayload struct {
	Transactions []*Transaction `json:"transactions"`
}

//TransactionResponseError struct to hold error response from api node
type TransactionResponseError struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
	Data         string `json:"data"`
}

//TransactionQueryParams for returing filtered list of transactions
type TransactionQueryParams struct {
	ID          string `url:"id,omitempty"`
	BlockID     string `url:"blockId,omitempty"`
	SenderID    string `url:"senderId,omitempty"`
	RecipientID string `url:"recipientId,omitempty"`
	Limit       int    `url:"limit,omitempty"`
	Offset      int    `url:"offset,omitempty"`
	OrderBy     string `url:"orderBy,omitempty"` //"Name of column to order. After column name must go 'desc' or 'asc' to choose order type, prefix for column name is t_. Example: orderBy=t_timestamp:desc (String)"
}

//TransactionResponse structure holds parsed jsong reply from ark-node
//when calling list methods the Transactions [] has results
//when calling get methods the transaction object (Single) has results
type TransactionResponse struct {
	Success      bool              `json:"success"`
	Transactions []TransactionData `json:"transactions"`
	Transaction  TransactionData   `json:"transaction"`
	Count        string            `json:"count"`
	Error        string            `json:"error"`
}

//TransactionData holds parsed Transaction data from rest json responses...
type TransactionData struct {
	ID              string `json:"id"`
	Blockid         string `json:"blockid"`
	Height          int    `json:"height"`
	Type            int    `json:"type"`
	Timestamp       int    `json:"timestamp"`
	Amount          int    `json:"amount"`
	Fee             int    `json:"fee"`
	VendorField     string `json:"vendorField"`
	SenderID        string `json:"senderId"`
	RecipientID     string `json:"recipientId"`
	SenderPublicKey string `json:"senderPublicKey"`
	Signature       string `json:"signature"`
	Asset           struct {
	} `json:"asset"`
	Confirmations int `json:"confirmations"`
}

//Error interface function
func (e TransactionResponseError) Error() string {
	return fmt.Sprintf("ArkServiceApi: %v %v", e.Success, e.ErrorMessage)
}

//PostTransaction to selected ARKNetwork
func (s *ArkClient) PostTransaction(tx *Transaction) (PostTransactionResponse, *http.Response, error) {
	respTr := new(PostTransactionResponse)
	errTr := new(TransactionResponseError)

	var payload transactionPayload
	payload.Transactions = append(payload.Transactions, tx)

	resp, err := s.sling.New().Post("peer/transactions").BodyJSON(payload).Receive(respTr, errTr)

	if err == nil {
		err = errTr
	}

	return *respTr, resp, err
}

//ListTransaction function returns list of peers from ArkNode
func (s *ArkClient) ListTransaction(params TransactionQueryParams) (TransactionResponse, *http.Response, error) {
	transactionResponse := new(TransactionResponse)
	transactionResponseErr := new(TransactionResponseError)
	resp, err := s.sling.New().Get("api/transactions").QueryStruct(&params).Receive(transactionResponse, transactionResponseErr)
	if err == nil {
		err = transactionResponseErr
	}

	return *transactionResponse, resp, err
}

//ListTransactionUnconfirmed function returns list of peers from ArkNode
func (s *ArkClient) ListTransactionUnconfirmed(params TransactionQueryParams) (TransactionResponse, *http.Response, error) {
	transactionResponse := new(TransactionResponse)
	transactionResponseErr := new(TransactionResponseError)
	resp, err := s.sling.New().Get("api/transactions/unconfirmed").QueryStruct(&params).Receive(transactionResponse, transactionResponseErr)
	if err == nil {
		err = transactionResponseErr
	}

	return *transactionResponse, resp, err
}

//GetTransaction function returns list of peers from ArkNode
func (s *ArkClient) GetTransaction(params TransactionQueryParams) (TransactionResponse, *http.Response, error) {
	transactionResponse := new(TransactionResponse)
	transactionResponseErr := new(TransactionResponseError)
	resp, err := s.sling.New().Get("api/transactions/get").QueryStruct(&params).Receive(transactionResponse, transactionResponseErr)
	if err == nil {
		err = transactionResponseErr
	}

	return *transactionResponse, resp, err
}

//GetTransactionUnconfirmed function returns list of peers from ArkNode
func (s *ArkClient) GetTransactionUnconfirmed(params TransactionQueryParams) (TransactionResponse, *http.Response, error) {
	transactionResponse := new(TransactionResponse)
	transactionResponseErr := new(TransactionResponseError)
	resp, err := s.sling.New().Get("api/transactions/unconfirmed/get").QueryStruct(&params).Receive(transactionResponse, transactionResponseErr)
	if err == nil {
		err = transactionResponseErr
	}

	return *transactionResponse, resp, err
}
