package db

import (
	"blockbook/bchain"
	"bytes"

	"io"
	"github.com/golang/glog"
	"github.com/martinboehm/btcd/wire"
	"github.com/juju/errors"
)

type WitnessAddressType struct {
	Version uint8
	WitnessProgram []byte
}
type RangeAmountPairType struct {
	WitnessAddress WitnessAddressType
	ValueSat big.Int
}

type AssetAllocationTupleType struct {
	Asset uint32
	WitnessAddress WitnessAddressType
}

type AssetAllocation struct {
	AssetAllocationTuple AssetAllocationTupleType
	ListSendingAllocationAmounts []RangeAmountPairType
}

func (a *WitnessAddressType) Deserialize(r io.Reader) {
	Version, err := binarySerializer.Uint8(r)
	if err != nil {
		return errors.New("rocksdb: WitnessAddressType Deserialize Version: error %v", err)
	}
	WitnessProgram, err := wire.ReadVarBytes(r, 0, 256)
	if err != nil {
		return errors.New("rocksdb: WitnessAddressType Deserialize WitnessProgram: error %v", err)
	}
}

func (m *WitnessAddressType) ToString() string {
	if m != nil {
		if len(m.WitnessProgram) <= 4 && m.WitnessProgram == "burn" {
			return "burn"
		}
		// Convert data to base32:
		conv, err := bech32.ConvertBits([]byte(m.WitnessProgram), 8, 5, true)
		if err != nil {
			fmt.Println("Error:", err)
			return ""
		}
		encoded, err := bech32.Encode("sys", conv)
		if err != nil {
			fmt.Println("Error:", err)
			return ""
		}
		return encoded
	}
	return ""
}

func (a *RangeAmountPairType) Deserialize(r io.Reader) {
	err := WitnessAddress.Deserialize(r)
	if err != nil {
		return err
	}
	valueSat, err := binarySerializer.Uint64(r, wire.littleEndian)
	if err != nil {
		return errors.New("rocksdb: WitnessAddressType Deserialize ValueSat: error %v", err)
	}
	ValueSat := big.NewInt(valueSat)
}
func (a *AssetAllocationTupleType) Deserialize(r io.Reader) {
	Asset, err := binarySerializer.Uint32(r, wire.littleEndian)
	if err != nil {
		return errors.New("rocksdb: AssetAllocationTupleType Deserialize Asset: error %v", err)
	}
	err = WitnessAddress.Deserialize(r)
	if err != nil {
		return err
	}
}
func (a *AssetAllocation) Deserialize(r io.Reader) {
	err := AssetAllocationTuple.Deserialize(r)
	if err != nil {
		return err
	}
	numReceivers, err := binarySerializer.Uint8(r)
	if err != nil {
		return errors.New("rocksdb: AssetAllocation Deserialize numReceivers: error %v", err)
	}
	ListSendingAllocationAmounts := make([]RangeAmountPairType, numReceivers)
	for _, allocation := range ListSendingAllocationAmount {
		err = allocation.Deserialize(r)
		if err != nil {
			return err
		}
	}
}

func (d *RocksDB) ConnectAssetAllocationOutput(sptData []byte, balances map[string]*AddrBalance, version int32) (*bchain.SyscoinOutputPackage, error) {
	r := bytes.NewReader(sptData)
	var assetAllocation AssetAllocation
	r.Seek(0, 0)
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return nil, err
	}
	totalAssetSentValue := big.NewInt(0)
	assetGuid := pt.AssetAllocationTuple.Asset
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(tx.AssetAllocationTuple.WitnessAddress.ToString())
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("rocksdb: asset sender addrDesc: %v error %v", assetSenderAddrDesc, err)
			}
		} else {
			glog.V(1).Infof("rocksdb: skipping asset sender addrDesc of length %d", len(assetSenderAddrDesc))
		}
		return nil, errors.New("Skipping asset sender")
	}
	strAddrDescriptors := make([]string, 0, len(tx.ListSendingAllocationAmount))
	for _, allocation := range tx.ListSendingAllocationAmount {
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(allocation.WitnessAddress.ToString())
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("rocksdb: asset addrDesc: %v error %v", addrDesc, err)
				}
			} else {
				glog.V(1).Infof("rocksdb: skipping asset addrDesc of length %d", len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		balance, e := balances[strAddrDesc]
		if !e {
			balance, err = d.GetAddrDescBalance(addrDesc, addressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			if balance == nil {
				balance = &AddrBalance{}
			}
			balances[strAddrDesc] = balance
		}
		if balance.BalanceAssetAllocatedSat == nil{
			balance.BalanceAssetAllocatedSat = map[uint32]big.Int{}
		}
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
		if !ok {
			balanceAssetAllocatedSat.Set(big.NewInt(0))
		}
		strAddrDescriptors = append(strAddrDescriptors, strAddrDesc)
		amount := allocation.ValueSat.AsBigInt()
		balanceAssetAllocatedSat.Add(&balanceAssetAllocatedSat, &amount)
		totalAssetSentValue.Add(totalAssetSentValue, &amount)
		balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
	}
	return &bchain.SyscoinOutputPackage{
		Version: version,
		AssetGuid: assetGuid,
		TotalAssetSentValue: *totalAssetSentValue,
		AssetSenderAddrDesc: assetSenderAddrDesc,
		AssetReceiverStrAddrDesc: strAddrDescriptors,
	}, nil
}
func (d *RocksDB) ConnectAssetAllocationInput(outputPackage *bchain.SyscoinOutputPackage, balance *AddrBalance) bool {
	
	if balance.SentAssetAllocatedSat == nil{
		balance.SentAssetAllocatedSat = map[uint32]big.Int{}
	}
	sentAssetAllocatedSat := balance.SentAssetAllocatedSat[outputPackage.AssetGuid]
	balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[outputPackage.AssetGuid]
	if !ok {
		balanceAssetAllocatedSat.Set(big.NewInt(0))
	}
	balanceAssetAllocatedSat.Sub(&balanceAssetAllocatedSat, &outputPackage.TotalAssetSentValue)
	sentAssetAllocatedSat.Add(&sentAssetAllocatedSat, &outputPackage.TotalAssetSentValue)
	if balanceAssetAllocatedSat.Sign() < 0 {
		d.resetValueSatToZero(&balanceAssetAllocatedSat, outputPackage.AssetSenderAddrDesc, "balance")
	}
	balance.SentAssetAllocatedSat[outputPackage.AssetGuid] = sentAssetAllocatedSat
	balance.BalanceAssetAllocatedSat[outputPackage.AssetGuid] = balanceAssetAllocatedSat
	return true

}
func (d *RocksDB) ConnectSyscoinOutputs(script []byte, balances map[string]*AddrBalance, version int32) (*bchain.SyscoinOutputPackage, error) {
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil, nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.ConnectAssetAllocationOutput(sptData, balances, version)
	}
	return nil, errors.New("Not supported OP")
}
func (d *RocksDB) ConnectSyscoinInputs(outputPackage *bchain.SyscoinOutputPackage, balance *AddrBalance) bool {
	if d.chainParser.IsAssetAllocationTx(outputPackage.Version) {
		return d.ConnectAssetAllocationInput(outputPackage, balance)
	}
	return false
}
