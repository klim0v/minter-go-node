package transaction

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"golang.org/x/crypto/sha3"
	"math/big"
)

// TxType of transaction is determined by a single byte.
type TxType byte

type SigType byte

const (
	TypeSend                   TxType = 0x01
	TypeSellCoin               TxType = 0x02
	TypeSellAllCoin            TxType = 0x03
	TypeBuyCoin                TxType = 0x04
	TypeCreateCoin             TxType = 0x05
	TypeDeclareCandidacy       TxType = 0x06
	TypeDelegate               TxType = 0x07
	TypeUnbond                 TxType = 0x08
	TypeRedeemCheck            TxType = 0x09
	TypeSetCandidateOnline     TxType = 0x0A
	TypeSetCandidateOffline    TxType = 0x0B
	TypeCreateMultisig         TxType = 0x0C
	TypeMultisend              TxType = 0x0D
	TypeEditCandidate          TxType = 0x0E
	TypeSetHaltBlock           TxType = 0x0F
	TypeRecreateCoin           TxType = 0x10
	TypeEditCoinOwner          TxType = 0x11
	TypeEditMultisig           TxType = 0x12
	TypePriceVote              TxType = 0x13
	TypeEditCandidatePublicKey TxType = 0x14
	TypeAddLiquidity           TxType = 0x15
	TypeRemoveLiquidity        TxType = 0x16
	TypeSellSwapPool           TxType = 0x17
	TypeBuySwapPool            TxType = 0x18
	TypeSellAllSwapPool        TxType = 0x19
	TypeEditCommission         TxType = 0x1A
	TypeMoveStake              TxType = 0x1B
	TypeMintToken              TxType = 0x1C
	TypeBurnToken              TxType = 0x1D
	TypeCreateToken            TxType = 0x1E
	TypeRecreateToken          TxType = 0x1F
	TypePriceCommission        TxType = 0x20
	TypeUpdateNetwork          TxType = 0x21

	SigTypeSingle SigType = 0x01
	SigTypeMulti  SigType = 0x02
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

var (
	CommissionMultiplier = big.NewInt(10e14)
)

type Transaction struct {
	Nonce         uint64
	ChainID       types.ChainID
	GasPrice      uint32
	GasCoin       types.CoinID
	Type          TxType
	Data          RawData
	Payload       []byte
	ServiceData   []byte
	SignatureType SigType
	SignatureData []byte

	decodedData Data
	sig         *Signature
	multisig    *SignatureMulti
	sender      *types.Address
}

type Signature struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

type SignatureMulti struct {
	Multisig   types.Address
	Signatures []Signature
}

type RawData []byte

type totalSpends []totalSpend

func (tss *totalSpends) Add(coin types.CoinID, value *big.Int) {
	for i, t := range *tss {
		if t.Coin == coin {
			(*tss)[i].Value.Add((*tss)[i].Value, big.NewInt(0).Set(value))
			return
		}
	}

	*tss = append(*tss, totalSpend{
		Coin:  coin,
		Value: big.NewInt(0).Set(value),
	})
}

type totalSpend struct {
	Coin  types.CoinID
	Value *big.Int
}

type conversion struct {
	FromCoin    types.CoinID
	FromAmount  *big.Int
	FromReserve *big.Int
	ToCoin      types.CoinID
	ToAmount    *big.Int
	ToReserve   *big.Int
}

type Data interface {
	String() string
	Gas() int64
	Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, coin types.CoinID, newInt *big.Int) Response
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(tx)
}

func (tx *Transaction) Gas() int64 {
	return tx.decodedData.Gas() + tx.payloadGas()
}

func (tx *Transaction) payloadGas() int64 {
	return int64(len(tx.Payload)+len(tx.ServiceData)) * commissions.PayloadByte
}

func (tx *Transaction) CommissionInBaseCoin() *big.Int {
	commissionInBaseCoin := big.NewInt(0).Mul(big.NewInt(int64(tx.GasPrice)), big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)

	return commissionInBaseCoin
}

func (tx *Transaction) String() string {
	sender, _ := tx.Sender()

	return fmt.Sprintf("TX nonce:%d from:%s payload:%s data:%s",
		tx.Nonce, sender.String(), tx.Payload, tx.decodedData.String())
}

func (tx *Transaction) Sign(prv *ecdsa.PrivateKey) error {
	h := tx.Hash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return err
	}

	tx.SetSignature(sig)

	return nil
}

func (tx *Transaction) SetSignature(sig []byte) {
	switch tx.SignatureType {
	case SigTypeSingle:
		{
			if tx.sig == nil {
				tx.sig = &Signature{}
			}

			tx.sig.R = new(big.Int).SetBytes(sig[:32])
			tx.sig.S = new(big.Int).SetBytes(sig[32:64])
			tx.sig.V = new(big.Int).SetBytes([]byte{sig[64] + 27})

			data, err := rlp.EncodeToBytes(tx.sig)

			if err != nil {
				panic(err)
			}

			tx.SignatureData = data
		}
	case SigTypeMulti:
		{
			if tx.multisig == nil {
				tx.multisig = &SignatureMulti{
					Multisig:   types.Address{},
					Signatures: []Signature{},
				}
			}

			tx.multisig.Signatures = append(tx.multisig.Signatures, Signature{
				V: new(big.Int).SetBytes([]byte{sig[64] + 27}),
				R: new(big.Int).SetBytes(sig[:32]),
				S: new(big.Int).SetBytes(sig[32:64]),
			})

			data, err := rlp.EncodeToBytes(tx.multisig)

			if err != nil {
				panic(err)
			}

			tx.SignatureData = data
		}
	}
}

func (tx *Transaction) Sender() (types.Address, error) {
	if tx.sender != nil {
		return *tx.sender, nil
	}

	switch tx.SignatureType {
	case SigTypeSingle:
		sender, err := RecoverPlain(tx.Hash(), tx.sig.R, tx.sig.S, tx.sig.V)
		if err != nil {
			return types.Address{}, err
		}

		tx.sender = &sender
		return sender, nil
	case SigTypeMulti:
		return tx.multisig.Multisig, nil
	}

	return types.Address{}, errors.New("unknown signature type")
}

func (tx *Transaction) Hash() types.Hash {
	return rlpHash([]interface{}{
		tx.Nonce,
		tx.ChainID,
		tx.GasPrice,
		tx.GasCoin,
		tx.Type,
		tx.Data,
		tx.Payload,
		tx.ServiceData,
		tx.SignatureType,
	})
}

func (tx *Transaction) SetDecodedData(data Data) {
	tx.decodedData = data
}

func (tx *Transaction) GetDecodedData() Data {
	return tx.decodedData
}

func (tx *Transaction) SetMultisigAddress(address types.Address) {
	if tx.multisig == nil {
		tx.multisig = &SignatureMulti{}
	}

	tx.multisig.Multisig = address

	data, err := rlp.EncodeToBytes(tx.multisig)

	if err != nil {
		panic(err)
	}

	tx.SignatureData = data
}

func RecoverPlain(sighash types.Hash, R, S, Vb *big.Int) (types.Address, error) {
	if Vb.BitLen() > 8 {
		return types.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, true) {
		return types.Address{}, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the snature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return types.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return types.Address{}, errors.New("invalid public key")
	}
	var addr types.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func rlpHash(x interface{}) (h types.Hash) {
	hw := sha3.NewLegacyKeccak256()
	err := rlp.Encode(hw, x)
	if err != nil {
		panic(err)
	}
	hw.Sum(h[:0])
	return h
}

func CheckForCoinSupplyOverflow(coin calculateCoin, delta *big.Int) *Response {
	total := big.NewInt(0).Set(coin.Volume())
	total.Add(total, delta)

	if total.Cmp(coin.MaxSupply()) == 1 {
		return &Response{
			Code: code.CoinSupplyOverflow,
			Log:  "Coin supply overflow",
			Info: EncodeError(code.NewCoinSupplyOverflow(delta.String(), coin.Volume().String(), total.String(), coin.MaxSupply().String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	return nil
}

func CheckReserveUnderflow(coin calculateCoin, delta *big.Int) *Response {
	total := big.NewInt(0).Sub(coin.Reserve(), delta)

	if total.Cmp(minCoinReserve) == -1 {
		min := big.NewInt(0).Add(minCoinReserve, delta)
		return &Response{
			Code: code.CoinReserveUnderflow,
			Log:  fmt.Sprintf("coin %s reserve is too small (%s, required at least %s)", coin.GetFullSymbol(), coin.Reserve().String(), min.String()),
			Info: EncodeError(code.NewCoinReserveUnderflow(delta.String(), coin.Reserve().String(), total.String(), minCoinReserve.String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	return nil
}
