package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-proxy-go/data"
	"github.com/ElrondNetwork/elrond-proxy-go/rosetta/client"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

type constructionAPIService struct {
	elrondClient *client.ElrondClient
}

// NewConstructionAPIService creates a new instance of an ConstructionAPIService.
func NewConstructionAPIService(elrondClient *client.ElrondClient) server.ConstructionAPIServicer {
	return &constructionAPIService{
		elrondClient: elrondClient,
	}
}

func checkOperationsAndMeta(ops []*types.Operation, meta map[string]interface{}) *types.Error {
	terr := ErrConstructionCheck
	if len(ops) == 0 {
		terr.Message += "invalid number of operations"
		return terr
	}

	for _, op := range ops {
		if !checkOperationsType(op) {
			terr.Message += "unsupported operation type"
			return terr
		}
	}

	if meta["gasLimit"] != nil {
		if _, ok := meta["gasLimit"].(uint64); ok {
			terr.Message += "invalid gas limit"
			return terr
		}
	}
	if meta["gasPrice"] != nil {
		if _, ok := meta["gasPrice"].(uint64); ok {
			terr.Message += "invalid gas price"
			return terr
		}
	}

	return nil
}

func checkOperationsType(op *types.Operation) bool {
	for _, supOp := range SupportedOperationTypes {
		if supOp == op.Type {
			return true
		}
	}

	return false
}

func getOptionsFromOperations(ops []*types.Operation) map[string]interface{} {
	options := make(map[string]interface{})
	options["sender"] = ops[0].Account.Address
	options["receiver"] = ops[1].Account.Address
	options["type"] = ops[0].Type
	options["value"] = ops[1].Amount.Value

	return options
}

func (s *constructionAPIService) ConstructionPreprocess(
	_ context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	if err := checkOperationsAndMeta(request.Operations, request.Metadata); err != nil {
		return nil, err
	}

	options := getOptionsFromOperations(request.Operations)

	if len(request.MaxFee) > 0 {
		maxFee := request.MaxFee[0]
		if maxFee.Currency.Symbol != ElrondCurrency.Symbol ||
			maxFee.Currency.Decimals != ElrondCurrency.Decimals {
			terr := ErrConstructionCheck
			terr.Message += "invalid currency"
			return nil, terr
		}

		options["maxFee"] = maxFee.Value
	}

	if request.SuggestedFeeMultiplier != nil {
		options["feeMultiplier"] = *request.SuggestedFeeMultiplier
	}

	if request.Metadata["gasLimit"] != nil {
		options["gasLimit"] = request.Metadata["gasLimit"]
	}
	if request.Metadata["gasPrice"] != nil {
		options["gasPrice"] = request.Metadata["gasPrice"]
	}
	if request.Metadata["data"] != nil {
		options["data"] = request.Metadata["data"]
	}

	return &types.ConstructionPreprocessResponse{
		Options: options,
	}, nil
}

func (s *constructionAPIService) ConstructionMetadata(
	_ context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	txType, ok := request.Options["type"].(string)
	if !ok {
		terr := ErrInvalidInputParam
		terr.Message += "transaction type"
		return nil, terr
	}

	networkConfig, err := s.elrondClient.GetNetworkConfig()
	if err != nil {
		return nil, ErrUnableToGetNetworkConfig
	}

	metadata, suggestedFee, errS := s.computeMetadataAndSuggestedFee(txType, request.Options, networkConfig)
	if errS != nil {
		return nil, errS
	}

	return &types.ConstructionMetadataResponse{
		Metadata: metadata,
		SuggestedFee: []*types.Amount{
			{
				Value:    suggestedFee,
				Currency: ElrondCurrency,
			},
		},
	}, nil
}

func (s *constructionAPIService) computeMetadataAndSuggestedFee(txType string, options objectsMap, networkConfig *client.NetworkConfig) (objectsMap, string, *types.Error) {
	metadata := make(objectsMap)

	if gasLimit, ok := options["gasLimit"]; ok {
		metadata["gasLimit"] = gasLimit
	} else {
		gasLimit, err := s.estimateGasLimit(txType, networkConfig)
		if err != nil {
			return nil, "", err
		}

		metadata["gasLimit"] = gasLimit
	}

	if gasPrice, ok := options["gasPrice"]; ok {
		metadata["gasPrice"] = gasPrice
	} else {
		metadata["gasPrice"] = networkConfig.MinGasPrice
	}

	if dataField, ok := options["data"]; ok {
		// convert string to byte array
		metadata["data"] = []byte(fmt.Sprintf("%v", dataField))
	}

	suggestedFee := big.NewInt(0).Mul(
		big.NewInt(0).SetUint64(getUint64Value(metadata["gasPrice"])),
		big.NewInt(0).SetUint64(getUint64Value(metadata["gasLimit"])),
	)

	var ok bool
	if metadata["sender"], ok = options["sender"]; !ok {
		return nil, "", ErrMalformedValue
	}

	if metadata["receiver"], ok = options["receiver"]; !ok {
		return nil, "", ErrMalformedValue
	}
	if metadata["value"], ok = options["value"]; !ok {
		return nil, "", ErrMalformedValue
	}

	metadata["chainID"] = networkConfig.ChainID
	metadata["version"] = networkConfig.MinTxVersion

	account, err := s.elrondClient.GetAccount(options["sender"].(string))
	if err != nil {
		return nil, "", ErrUnableToGetAccount
	}

	metadata["nonce"] = account.Nonce

	return metadata, suggestedFee.String(), nil
}

func getUint64Value(obj interface{}) uint64 {
	if value, ok := obj.(uint64); ok {
		return value
	}
	if value, ok := obj.(float64); ok {
		return uint64(value)
	}

	return 0
}

func (s *constructionAPIService) estimateGasLimit(operationType string, networkConfig *client.NetworkConfig) (uint64, *types.Error) {
	switch operationType {
	case opTransfer:
		return networkConfig.MinGasLimit, nil
	default:
		return 0, ErrNotImplemented
	}
}

func (s *constructionAPIService) ConstructionPayloads(
	_ context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	erdTx, err := createTransaction(request)
	if err != nil {
		return nil, ErrMalformedValue
	}

	mtx, err := json.Marshal(erdTx)
	if err != nil {
		return nil, ErrMalformedValue
	}

	unsignedTx := hex.EncodeToString(mtx)

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: unsignedTx,
		Payloads: []*types.SigningPayload{
			{
				AccountIdentifier: &types.AccountIdentifier{
					Address: request.Operations[0].Account.Address,
				},
				SignatureType: types.Ed25519,
				Bytes:         mtx,
			},
		},
	}, nil
}

func (s *constructionAPIService) ConstructionParse(
	_ context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	elrondTx, err := getTxFromRequest(request.Transaction)
	if err != nil {
		return nil, ErrMalformedValue
	}

	var signers []*types.AccountIdentifier
	if request.Signed {
		signers = []*types.AccountIdentifier{
			{
				Address: elrondTx.Sender,
			},
		}
	}

	return &types.ConstructionParseResponse{
		Operations:               createOperationsFromPreparedTx(elrondTx),
		AccountIdentifierSigners: signers,
	}, nil
}

func createTransaction(request *types.ConstructionPayloadsRequest) (*data.Transaction, error) {
	tx := &data.Transaction{}

	requestMetadataBytes, err := json.Marshal(request.Metadata)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(requestMetadataBytes, tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func getTxFromRequest(txString string) (*data.Transaction, error) {
	txBytes, err := hex.DecodeString(txString)
	if err != nil {
		return nil, err
	}

	var elrondTx data.Transaction
	err = json.Unmarshal(txBytes, &elrondTx)
	if err != nil {
		return nil, err
	}

	return &elrondTx, nil
}

func (s *constructionAPIService) ConstructionCombine(
	_ context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	elrondTx, err := getTxFromRequest(request.UnsignedTransaction)
	if err != nil {
		return nil, ErrMalformedValue
	}

	if len(request.Signatures) != 1 {
		return nil, ErrInvalidInputParam
	}

	// is this the right signature
	txSignature := hex.EncodeToString(request.Signatures[0].Bytes)
	elrondTx.Signature = txSignature

	signedTxBytes, err := json.Marshal(elrondTx)
	if err != nil {
		return nil, ErrMalformedValue
	}

	signedTx := hex.EncodeToString(signedTxBytes)

	return &types.ConstructionCombineResponse{
		SignedTransaction: signedTx,
	}, nil
}

func (s *constructionAPIService) ConstructionDerive(
	_ context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if request.PublicKey.CurveType != types.Edwards25519 {
		return nil, ErrUnsupportedCurveType
	}

	pubKey := request.PublicKey.Bytes
	address, err := s.elrondClient.EncodeAddress(pubKey)
	if err != nil {
		return nil, ErrMalformedValue
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: address,
		},
		Metadata: nil,
	}, nil
}

func (s *constructionAPIService) ConstructionHash(
	_ context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	elrondTx, err := getTxFromRequest(request.SignedTransaction)
	if err != nil {
		return nil, ErrMalformedValue
	}

	txHash, err := s.elrondClient.SimulateTx(elrondTx)
	if err != nil {
		return nil, ErrMalformedValue
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txHash,
		},
	}, nil
}

func (s *constructionAPIService) ConstructionSubmit(
	_ context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	elrondTx, err := getTxFromRequest(request.SignedTransaction)
	if err != nil {
		return nil, ErrMalformedValue
	}

	txHash, err := s.elrondClient.SendTx(elrondTx)
	if err != nil {
		return nil, ErrMalformedValue
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txHash,
		},
	}, nil
}
