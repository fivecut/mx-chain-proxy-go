package block

import "github.com/ElrondNetwork/elrond-proxy-go/data"

// FacadeHandler interface defines methods that can be used from `elrondProxyFacade` context variable
type FacadeHandler interface {
	GetHighestBlockNonce() (uint64, error)
	GetBlockByNonce(nonce uint64) (data.ApiBlock, error)
}
