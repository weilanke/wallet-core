package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dabankio/wallet-core/bip44"
	"github.com/dabankio/wallet-core/core"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/lomocoin/gobbc"
	"github.com/pkg/errors"
)

const (
	symbol = "BBC"
)

type BBC struct {
	Symbol         string
	DerivationPath accounts.DerivationPath
	MasterKey      *ExtendedKey
}

var _ core.Coin = (*BBC)(nil) //type ensure

// NewCoin new bbc coin implementation
func NewCoin(seed []byte) (core.Coin, error) {
	return NewCoinWithPath(seed, bip44.PathFormat)
}

// NewCoinWithPath new bbc coin implementation, 只推导1个地址
func NewCoinWithPath(seed []byte, path string) (core.Coin, error) {
	if strings.Count(path, "%d") != 1 {
		return nil, errors.New("path 应包含且仅且包含1个%d占位符")
	}
	bbcBip44ID, err := bip44.GetCoinType(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get BBC bip44 id")
	}
	c := new(BBC)
	c.Symbol = symbol
	c.DerivationPath, err = accounts.ParseDerivationPath(fmt.Sprintf(path, bbcBip44ID))
	if err != nil {
		return nil, errors.Wrap(err, "bip44.GetCoinDerivationPath err:")
	}
	c.MasterKey, err = NewMaster(seed)
	if err != nil {
		return nil, errors.Wrap(err, "unable to new master key for bbc")
	}
	return c, nil
}

// NewCoinFullPath new bbc coin implementation, with full bip44 path
func NewCoinFullPath(seed []byte, accountIndex, changeType, index int) (core.Coin, error) {
	var err error
	c := new(BBC)
	c.Symbol = symbol
	c.DerivationPath, err = bip44.GetFullCoinDerivationPath(symbol, accountIndex, changeType, index)
	if err != nil {
		return nil, errors.Wrap(err, "bip44.GetFullCoinDerivationPath err:")
	}
	c.MasterKey, err = NewMaster(seed)
	if err != nil {
		return nil, errors.Wrap(err, "unable to new master key for bbc")
	}
	return c, nil
}

// DeriveAddress derives the account address of the derivation path.
func (c *BBC) DeriveAddress() (address string, err error) {
	child, err := c.derive()
	if err != nil {
		return "", err
	}
	pubk, err := gobbc.Seed2pubkString(child.key)
	if err != nil {
		return "", err
	}
	return gobbc.GetPubKeyAddress(pubk)
}

// DerivePublicKey derives the public key of the derivation path.
func (c *BBC) DerivePublicKey() (publicKey string, err error) {
	child, err := c.derive()
	if err != nil {
		return "", err
	}
	return gobbc.Seed2pubkString(child.key)
}

func (c *BBC) derive() (*ExtendedKey, error) {
	var err error
	childKey := c.MasterKey
	for _, childNum := range c.DerivationPath {
		childKey, err = childKey.Child(childNum)
		if err != nil {
			return nil, err
		}
	}
	return childKey, nil
}

// DerivePrivateKey derives the private key of the derivation path.
func (c *BBC) DerivePrivateKey() (privateKey string, err error) {
	child, err := c.derive()
	if err != nil {
		return "", err
	}
	return gobbc.Seed2string(child.key), nil
}

// DecodeTx decodes raw tx to human readable format
func (c *BBC) DecodeTx(msg string) (string, error) {
	tx, err := gobbc.DecodeRawTransaction(msg, false)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(tx)
	if err != nil {
		return "", errors.Wrapf(err, "unable to marshal json, %#v", tx)
	}
	return string(b), nil
}

// Sign signs raw tx with privateKey
// 首先尝试解析为带模版数据的待签数据，无法解析则尝试一般原始交易
func (c *BBC) Sign(msg, privateKey string) (string, error) {
	var err error
	// 1尝试解析为多签数据
	if txData := tryParseTxDataWithTemplate(msg); txData != nil {
		txData.TxHex, err = c.SignTemplate(txData.TxHex, txData.TplHex, privateKey)
		if err != nil {
			return msg, errors.Wrap(err, "failed to encode tx")
		}
		return txData.EncodeString()
	}

	// 2无法解析为带模版待签数据则认为是原始交易
	return c.SignTemplate(msg, "", privateKey)
}

// SignTemplate signs raw tx with privateKey
func (c *BBC) SignTemplate(msg, templateData, privateKey string) (sig string, err error) {
	//尝试解析为原始交易
	tx, err := gobbc.DecodeRawTransaction(msg, true)
	if err != nil {
		return msg, errors.Wrap(err, "unable to parse tx data")
	}

	err = tx.SignWithPrivateKey(templateData, privateKey)
	if err != nil {
		return msg, errors.Wrap(err, "sign failed")
	}
	return tx.Encode(true)
}

// VerifySignature verifies rawTx's signature is intact
func (c *BBC) VerifySignature(pubKey, msg, signature string) error {
	return errors.New("verify signature not supported for BBC currently")
}

func tryParseTxDataWithTemplate(msg string) *gobbc.TXData {
	var data gobbc.TXData
	if err := data.DecodeString(msg); err != nil {
		return nil
	}
	return &data
}
