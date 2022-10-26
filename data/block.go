package data

import (
	"github.com/ElrondNetwork/elrond-go-core/data/api"
)

// AtlasBlock is a block, as required by BlockAtlas
// Will be removed when using the "hyperblock" route in BlockAtlas as well.
type AtlasBlock struct {
	Nonce        uint64                `form:"nonce" json:"nonce"`
	Hash         string                `form:"hash" json:"hash"`
	Transactions []DatabaseTransaction `form:"transactions" json:"transactions"`
}

// BlockApiResponse is a response holding a block
type BlockApiResponse struct {
	Data  BlockApiResponsePayload `json:"data"`
	Error string                  `json:"error"`
	Code  ReturnCode              `json:"code"`
}

// BlockApiResponsePayload wraps a block
type BlockApiResponsePayload struct {
	Block api.Block `json:"block"`
}

// HyperblockApiResponse is a response holding a hyperblock
type HyperblockApiResponse struct {
	Data  HyperblockApiResponsePayload `json:"data"`
	Error string                       `json:"error"`
	Code  ReturnCode                   `json:"code"`
}

// NewHyperblockApiResponse creates a HyperblockApiResponse
func NewHyperblockApiResponse(hyperblock api.Hyperblock) *HyperblockApiResponse {
	return &HyperblockApiResponse{
		Data: HyperblockApiResponsePayload{
			Hyperblock: hyperblock,
		},
		Code: ReturnCodeSuccess,
	}
}

// HyperblockApiResponsePayload wraps a hyperblock
type HyperblockApiResponsePayload struct {
	Hyperblock api.Hyperblock `json:"hyperblock"`
}

// InternalBlockApiResponse is a response holding an internal block
type InternalBlockApiResponse struct {
	Data  InternalBlockApiResponsePayload `json:"data"`
	Error string                          `json:"error"`
	Code  ReturnCode                      `json:"code"`
}

// InternalBlockApiResponsePayload wraps a internal generic block
type InternalBlockApiResponsePayload struct {
	Block interface{} `json:"block"`
}

// InternalMiniBlockApiResponse is a response holding an internal miniblock
type InternalMiniBlockApiResponse struct {
	Data  InternalMiniBlockApiResponsePayload `json:"data"`
	Error string                              `json:"error"`
	Code  ReturnCode                          `json:"code"`
}

// InternalMiniBlockApiResponsePayload wraps an internal miniblock
type InternalMiniBlockApiResponsePayload struct {
	MiniBlock interface{} `json:"miniblock"`
}
