package db

import (
	"blockbook/bchain"
	"bytes"
	"math/big"
	"encoding/hex"
	"github.com/golang/glog"
	"github.com/syscoin/btcd/wire"
	"github.com/juju/errors"
)


func (d *RocksDB) ConnectAssetAllocationOutput(sptData []byte, balances map[string]*AddrBalance, version int32, addresses addressesMap, btxID []byte, outputIndex int32) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocation
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return err
	}
	
	totalAssetSentValue := big.NewInt(0)
	assetGuid := assetAllocation.AssetAllocationTuple.Asset
	senderStr := assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderStr)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("rocksdb: asset %v sender addrDesc: %v error %v", assetGuid, senderStr, err)
			}
		} else {
			glog.V(1).Infof("rocksdb: skipping asset %v sender addrDesc: %v of length %d", assetGuid, senderStr, len(assetSenderAddrDesc))
		}
		return errors.New("Skipping asset sender")
	}
	for _, allocation := range assetAllocation.ListSendingAllocationAmounts {
		receiverStr := allocation.WitnessAddress.ToString("sys")
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverStr)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("rocksdb: asset %v addrDesc: %v witness program %v error %v", assetGuid, receiverStr,hex.EncodeToString(allocation.WitnessAddress.WitnessProgram), err)
				}
			} else {
				glog.V(1).Infof("rocksdb: skipping asset %v addrDesc of length %d", assetGuid, len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		_, exist := addresses[strAddrDesc]
		if !exist {
			addresses[strAddrDesc] = struct{}{}
		}
		balance, e := balances[strAddrDesc]
		if !e {
			balance, err = d.GetAddrDescBalance(addrDesc, addressBalanceDetailUTXOIndexed)
			if err != nil {
				return err
			}
			if balance == nil {
				balance = &AddrBalance{}
			}
			balances[strAddrDesc] = balance
			d.cbs.balancesMiss++
		} else {
			d.cbs.balancesHit++
		}

		// for each address returned, add it to map
		counted := addToAddressesMap(addresses, strAddrDesc, btxID, outputIndex)
		if !counted {
			balance.Txs++
		}

		if balance.BalanceAssetAllocatedSat == nil{
			balance.BalanceAssetAllocatedSat = map[uint32]big.Int{}
		}
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
		if !ok {
			balanceAssetAllocatedSat.Set(big.NewInt(0))
		}
		amount := big.NewInt(allocation.ValueSat)
		balanceAssetAllocatedSat.Add(&balanceAssetAllocatedSat, amount)
		totalAssetSentValue.Add(totalAssetSentValue, amount)
		balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
	}
	return d.ConnectAssetAllocationInput(assetGuid, totalAssetSentValue, assetSenderAddrDesc, balances)
}

func (d *RocksDB) DisconnectAssetAllocationOutput(sptData []byte, balances map[string]*AddrBalance, version int32, addresses addressesMap, btxID []byte, outputIndex int32) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocation
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, addressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			balances[s] = b
		}
		return b, nil
	}
	totalAssetSentValue := big.NewInt(0)
	assetGuid := assetAllocation.AssetAllocationTuple.Asset
	senderStr := assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderStr)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("rocksdb: asset %v sender addrDesc: %v error %v", assetGuid, senderStr, err)
			}
		} else {
			glog.V(1).Infof("rocksdb: skipping asset %v sender addrDesc: %v of length %d", assetGuid, senderStr, len(assetSenderAddrDesc))
		}
		return errors.New("Skipping asset sender")
	}
	for _, allocation := range assetAllocation.ListSendingAllocationAmounts {
		receiverStr := allocation.WitnessAddress.ToString("sys")
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverStr)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("rocksdb: asset %v addrDesc: %v witness program %v error %v", assetGuid, receiverStr,hex.EncodeToString(allocation.WitnessAddress.WitnessProgram), err)
				}
			} else {
				glog.V(1).Infof("rocksdb: skipping asset %v addrDesc of length %d", assetGuid, len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		_, exist := addresses[strAddrDesc]
		if !exist {
			addresses[strAddrDesc] = struct{}{}
		}
		balance, err = getAddressBalance(addrDesc)
		if err != nil {
			return err
		}
		if balance != nil {
			// subtract number of txs only once
			if !exist {
				balance.Txs--
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
			glog.Warningf("Balance for asset address %s (%s) not found", ad, addrDesc)
		}

		if balance.BalanceAssetAllocatedSat != nil{
			balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
			if ok {
				amount := big.NewInt(allocation.ValueSat)
				balanceAssetAllocatedSat.Sub(&balanceAssetAllocatedSat, amount)
				if balanceAssetAllocatedSat.Sign() < 0 {
					d.resetValueSatToZero(&balanceAssetAllocatedSat, addrDesc, "balance")
				}
				totalAssetSentValue.Add(totalAssetSentValue, amount)
				balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
			} else {
				glog.Warningf("Asset Balance for guid %s (%s) not found", assetGuid, addrDesc)
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
			glog.Warningf("Asset Balance for asset address %s (%s) not found", ad, addrDesc)
		}
	}
	return d.DisconnectAssetAllocationInput(assetGuid, totalAssetSentValue, assetSenderAddrDesc, balances)
}

func (d *RocksDB) ConnectAssetAllocationInput(assetGuid uint32, totalAssetSentValue big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*AddrBalance) error {
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		balance, err = d.GetAddrDescBalance(addrassetSenderAddrDescDesc, addressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &AddrBalance{}
		}
		balances[assetStrSenderAddrDesc] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}
	if balance.SentAssetAllocatedSat == nil{
		balance.SentAssetAllocatedSat = map[uint32]big.Int{}
	}
	sentAssetAllocatedSat := balance.SentAssetAllocatedSat[assetGuid]
	balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
	if !ok {
		balanceAssetAllocatedSat.Set(big.NewInt(0))
	}
	balanceAssetAllocatedSat.Sub(&balanceAssetAllocatedSat, &totalAssetSentValue)
	sentAssetAllocatedSat.Add(&sentAssetAllocatedSat, &totalAssetSentValue)
	if balanceAssetAllocatedSat.Sign() < 0 {
		d.resetValueSatToZero(&balanceAssetAllocatedSat, assetSenderAddrDesc, "balance")
	}
	balance.SentAssetAllocatedSat[assetGuid] = sentAssetAllocatedSat
	balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
	return nil

}

func (d *RocksDB) DisconnectAssetAllocationInput(assetGuid uint32, totalAssetSentValue big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*AddrBalance) error {
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		balance, err = d.GetAddrDescBalance(addrassetSenderAddrDescDesc, addressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			return errors.New("Asset Balance for sender address not found")
		}
		balances[assetStrSenderAddrDesc] = balance
	}
	if balance.SentAssetAllocatedSat != nil{
		sentAssetAllocatedSat := balance.SentAssetAllocatedSat[assetGuid]
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
		if ok {	
			balanceAssetAllocatedSat.Add(&balanceAssetAllocatedSat, &totalAssetSentValue)
			sentAssetAllocatedSat.Sub(&sentAssetAllocatedSat, &totalAssetSentValue)
			if sentAssetAllocatedSat.Sign() < 0 {
				d.resetValueSatToZero(&sentAssetAllocatedSat, assetSenderAddrDesc, "balance")
			}
			balance.SentAssetAllocatedSat[assetGuid] = sentAssetAllocatedSat
			balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
		} else {
			glog.Warningf("Asset Sender Balance for guid %s (%s) not found", assetGuid, assetStrSenderAddrDesc)
		}
	} else {
		glog.Warningf("Asset Balance for sender asset address %s not found", assetStrSenderAddrDesc)
	}
	return nil

}
func (d *RocksDB) ConnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*AddrBalance, version int32, addresses addressesMap, btxID []byte, outputIndex int32) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		glog.Warningf("rocksdb: asset addrDesc: height %d, tx %v, output %v, error %v", block.Height, tx.Txid, output, err)
		continue
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.ConnectAssetAllocationOutput(sptData, balances, version, addresses, btxID, outputIndex)
	}
	return nil
}

func (d *RocksDB) DisconnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*AddrBalance, version int32, addresses addressesMap, btxID []byte, outputIndex int32) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		glog.Warningf("rocksdb: disconnect asset addrDesc: height %d, tx %v, output %v, error %v", block.Height, tx.Txid, output, err)
		continue
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.DisconnectAssetAllocationOutput(sptData, balances, version, addresses, btxID, outputIndex)
	}
	return nil
}
