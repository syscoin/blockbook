package db

import (
	"blockbook/bchain"
	"bytes"
	"math/big"
	"github.com/golang/glog"
	"github.com/syscoin/btcd/wire"
	"github.com/juju/errors"
	"encoding/hex"
)

func (d *RocksDB) ConnectAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32) error {
	r := bytes.NewReader(sptData)
	var asset wire.AssetType
	err := asset.Deserialize(r)
	if err != nil {
		return err
	}
	assetGuid := asset.Asset
	senderStr := asset.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderStr)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("ConnectAssetOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, senderStr, err)
			}
		} else {
			glog.Warningf("ConnectAssetOutput sender with asset %v (%v) has invalid length: %d", assetGuid, senderStr, len(assetSenderAddrDesc))
		}
		return errors.New("ConnectAssetOutput Skipping asset tx")
	}
	balance, e := balances[senderStr]
	if !e {
		balance, err = d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &bchain.AddrBalance{}
		}
		balances[senderStr] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}

	// for each address returned, add it to map
	counted := addToAddressesMap(addresses, senderStr, btxID, outputIndex)
	if !counted {
		balance.Txs++
	}

	if len(asset.WitnessAddressTransfer.WitnessProgram) > 0 {
		transferStr := asset.WitnessAddressTransfer.ToString("sys")
		assetTransferWitnessAddrDesc, err := d.chainParser.GetAddrDescFromAddress(transferStr)
		if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("ConnectAssetOutput transferee with asset %v (%v) could not be decoded error %v", assetGuid, transferStr, err)
				}
			} else {
				glog.Warningf("ConnectAssetOutput transferee with asset %v (%v) has invalid length: %d", assetGuid, transferStr, len(assetTransferWitnessAddrDesc))
			}
			return errors.New("ConnectAssetOutput Skipping asset transfer tx")
		}
		balanceTransfer, e := balances[transferStr]
		if !e {
			balanceTransfer, err = d.GetAddrDescBalance(assetTransferWitnessAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return err
			}
			if balance == nil {
				balanceTransfer = &bchain.AddrBalance{}
			}
			balances[transferStr] = balanceTransfer
			d.cbs.balancesMiss++
		} else {
			d.cbs.balancesHit++
		}
		counted := addToAddressesMap(addresses, transferStr, btxID, outputIndex)
		if !counted {
			balanceTransfer.Txs++
		}
		// transfer balance from old address to transfered address
		if balance.BalanceAssetUnAllocatedSat == nil{
			balance.BalanceAssetUnAllocatedSat = map[uint32]big.Int{}
		}
		balanceTransfer.BalanceAssetUnAllocatedSat[assetGuid] = balance.BalanceAssetUnAllocatedSat[assetGuid]
		balance.BalanceAssetUnAllocatedSat[assetGuid] = *big.NewInt(0)
	} else {
		if balance.BalanceAssetUnAllocatedSat == nil{
			balance.BalanceAssetUnAllocatedSat = map[uint32]big.Int{}
		}
		balanceAssetUnAllocatedSat, ok := balance.BalanceAssetUnAllocatedSat[assetGuid]
		if !ok {
			balanceAssetUnAllocatedSat.Set(big.NewInt(0))
		}
		valueSat := big.NewInt(asset.Balance)
		balanceAssetUnAllocatedSat.Add(&balanceAssetUnAllocatedSat, valueSat)
		balance.BalanceAssetUnAllocatedSat[assetGuid] = balanceAssetUnAllocatedSat
		if assetGuid == 135354521 {
			glog.Warningf("asset tx %v assetGuid balance", assetGuid, balance.BalanceAssetUnAllocatedSat[assetGuid])
		}
	}
	return nil
}

func (d *RocksDB) ConnectAssetAllocationOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocationType
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
				glog.Warningf("ConnectAssetAllocationOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, senderStr, err)
			}
		} else {
			glog.Warningf("ConnectAssetAllocationOutput sender with asset %v (%v) has invalid length: %d", assetGuid, senderStr, len(assetSenderAddrDesc))
		}
		return errors.New("ConnectAssetAllocationOutput Skipping asset allocation tx")
	}
	for _, allocation := range assetAllocation.ListSendingAllocationAmounts {
		receiverStr := allocation.WitnessAddress.ToString("sys")
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverStr)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("ConnectAssetAllocationOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, receiverStr, err)
				}
			} else {
				glog.Warningf("ConnectAssetAllocationOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, receiverStr, len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		balance, e := balances[strAddrDesc]
		if !e {
			balance, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return err
			}
			if balance == nil {
				balance = &bchain.AddrBalance{}
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

		if balance.BalanceAssetAllocatedSat == nil {
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
	return d.ConnectAssetAllocationInput(btxID, assetGuid, version, totalAssetSentValue, assetSenderAddrDesc, balances)
}

func (d *RocksDB) DisconnectAssetAllocationOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocationType
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*bchain.AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
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
				glog.Warningf("DisconnectAssetAllocationOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, senderStr, err)
			}
		} else {
			glog.Warningf("DisconnectAssetAllocationOutput sender with asset %v (%v) has invalid length: %d", assetGuid, senderStr, len(assetSenderAddrDesc))
		}
		return errors.New("DisconnectAssetAllocationOutput Skipping disconnect asset allocation tx")
	}
	for _, allocation := range assetAllocation.ListSendingAllocationAmounts {
		receiverStr := allocation.WitnessAddress.ToString("sys")
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverStr)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("DisconnectAssetAllocationOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, receiverStr, err)
				}
			} else {
				glog.Warningf("DisconnectAssetAllocationOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, receiverStr, len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		_, exist := addresses[strAddrDesc]
		if !exist {
			addresses[strAddrDesc] = struct{}{}
		}
		balance, err := getAddressBalance(addrDesc)
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
			glog.Warningf("DisconnectAssetAllocationOutput Balance for asset address %s (%s) not found", ad, addrDesc)
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
				glog.Warningf("DisconnectAssetAllocationOutput Asset Balance for guid %s (%s) not found", assetGuid, addrDesc)
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
			glog.Warningf("DisconnectAssetAllocationOutput Asset Balance for asset address %s (%s) not found", ad, addrDesc)
		}
	}
	return d.DisconnectAssetAllocationInput(assetGuid, version, totalAssetSentValue, assetSenderAddrDesc, balances)
}

func (d *RocksDB) ConnectAssetAllocationInput(btxID []byte, assetGuid uint32, version int32, totalAssetSentValue *big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance) error {
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		balance, err := d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &bchain.AddrBalance{}
		}
		balances[assetStrSenderAddrDesc] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}
	if d.chainParser.IsSyscoinAssetSend(version) {
		if balance.SentAssetUnAllocatedSat == nil {
			balance.SentAssetUnAllocatedSat = map[uint32]big.Int{}
		}
		sentAssetUnAllocatedSat := balance.SentAssetUnAllocatedSat[assetGuid]
		balanceAssetUnAllocatedSat, ok := balance.BalanceAssetUnAllocatedSat[assetGuid]
		if !ok {
			balanceAssetUnAllocatedSat.Set(big.NewInt(0))
		}
		balanceAssetUnAllocatedSat.Sub(&balanceAssetUnAllocatedSat, totalAssetSentValue)
		sentAssetUnAllocatedSat.Add(&sentAssetUnAllocatedSat, totalAssetSentValue)
		if balanceAssetUnAllocatedSat.Sign() < 0 {
			glog.Warningf("ConnectAssetAllocationInput asset send negative assetguid %v txid %v", assetGuid, hex.EncodeToString(btxID))
			d.resetValueSatToZero(&balanceAssetUnAllocatedSat, assetSenderAddrDesc, "balance")
		}
		balance.SentAssetUnAllocatedSat[assetGuid] = sentAssetUnAllocatedSat
		balance.BalanceAssetUnAllocatedSat[assetGuid] = balanceAssetUnAllocatedSat
	} else {
		if balance.SentAssetAllocatedSat == nil {
			balance.SentAssetAllocatedSat = map[uint32]big.Int{}
		}
		sentAssetAllocatedSat := balance.SentAssetAllocatedSat[assetGuid]
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
		if !ok {
			balanceAssetAllocatedSat.Set(big.NewInt(0))
		}
		balanceAssetAllocatedSat.Sub(&balanceAssetAllocatedSat, totalAssetSentValue)
		sentAssetAllocatedSat.Add(&sentAssetAllocatedSat, totalAssetSentValue)
		if balanceAssetAllocatedSat.Sign() < 0 {
			d.resetValueSatToZero(&balanceAssetAllocatedSat, assetSenderAddrDesc, "balance")
		}
		balance.SentAssetAllocatedSat[assetGuid] = sentAssetAllocatedSat
		balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
	}
	return nil

}

func (d *RocksDB) DisconnectAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	r := bytes.NewReader(sptData)
	var asset wire.AssetType
	err := asset.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*bchain.AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			balances[s] = b
		}
		return b, nil
	}
	assetGuid := asset.Asset
	senderStr := asset.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderStr)
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	_, exist := addresses[assetStrSenderAddrDesc]
	if !exist {
		addresses[assetStrSenderAddrDesc] = struct{}{}
	}
	balance, err := getAddressBalance(assetSenderAddrDesc)
	if err != nil {
		return err
	}
	if balance != nil {
		// subtract number of txs only once
		if !exist {
			balance.Txs--
		}
	} else {
		ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(assetSenderAddrDesc)
		glog.Warningf("DisconnectAssetOutput Balance for asset address %s (%s) not found", ad, assetSenderAddrDesc)
	}
	if len(asset.WitnessAddressTransfer.WitnessProgram) > 0 {
		transferStr := asset.WitnessAddressTransfer.ToString("sys")
		assetTransferWitnessAddrDesc, err := d.chainParser.GetAddrDescFromAddress(transferStr)
		_, exist := addresses[assetStrSenderAddrDesc]
		if !exist {
			addresses[assetStrSenderAddrDesc] = struct{}{}
		}
		balanceTransfer, err := getAddressBalance(assetTransferWitnessAddrDesc)
		if err != nil {
			return err
		}
		if balanceTransfer != nil {
			// subtract number of txs only once
			if !exist {
				balanceTransfer.Txs--
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(assetTransferWitnessAddrDesc)
			glog.Warningf("DisconnectAssetOutput Balance for transfer asset address %s (%s) not found", ad, assetTransferWitnessAddrDesc)
		}
		// transfer values back to original owner and 0 out the
		balance.BalanceAssetUnAllocatedSat[assetGuid] = balanceTransfer.BalanceAssetUnAllocatedSat[assetGuid]
		balanceTransfer.BalanceAssetUnAllocatedSat[assetGuid] = *big.NewInt(0)
	} else if balance.SentAssetUnAllocatedSat != nil {
		sentAssetUnAllocatedSat := balance.SentAssetUnAllocatedSat[assetGuid]
		balanceAssetUnAllocatedSat, ok := balance.BalanceAssetUnAllocatedSat[assetGuid]
		if ok {
			valueSat := big.NewInt(asset.Balance)
			balanceAssetUnAllocatedSat.Add(&balanceAssetUnAllocatedSat, valueSat)
			sentAssetUnAllocatedSat.Sub(&sentAssetUnAllocatedSat, valueSat)
			if sentAssetUnAllocatedSat.Sign() < 0 {
				d.resetValueSatToZero(&sentAssetUnAllocatedSat, assetSenderAddrDesc, "balance")
			}
			balance.SentAssetUnAllocatedSat[assetGuid] = sentAssetUnAllocatedSat
			balance.BalanceAssetUnAllocatedSat[assetGuid] = balanceAssetUnAllocatedSat
		} else {
			glog.Warningf("DisconnectAssetOutput: Asset balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
		}
	} else {
		glog.Warningf("DisconnectAssetOutput: Asset Sent balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
	}
	return nil

}
func (d *RocksDB) DisconnectAssetAllocationInput(assetGuid uint32, version int32, totalAssetSentValue *big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance) error {
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		balance, err := d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			return errors.New("DisconnectAssetAllocationInput Asset Balance for sender address not found")
		}
		balances[assetStrSenderAddrDesc] = balance
	}
	if d.chainParser.IsSyscoinAssetSend(version) {
		if balance.SentAssetUnAllocatedSat != nil {
			sentAssetUnAllocatedSat := balance.SentAssetUnAllocatedSat[assetGuid]
			balanceAssetUnAllocatedSat, ok := balance.BalanceAssetUnAllocatedSat[assetGuid]
			if ok {	
				balanceAssetUnAllocatedSat.Add(&balanceAssetUnAllocatedSat, totalAssetSentValue)
				sentAssetUnAllocatedSat.Sub(&sentAssetUnAllocatedSat, totalAssetSentValue)
				if sentAssetUnAllocatedSat.Sign() < 0 {
					d.resetValueSatToZero(&sentAssetUnAllocatedSat, assetSenderAddrDesc, "balance")
				}
				balance.SentAssetUnAllocatedSat[assetGuid] = sentAssetUnAllocatedSat
				balance.BalanceAssetUnAllocatedSat[assetGuid] = balanceAssetUnAllocatedSat
			} else {
				glog.Warningf("DisconnectAssetAllocationInput: AssetSend UnAllocated balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
			}
		} else {
			glog.Warningf("DisconnectAssetAllocationInput: AssetSend SentUnAllocated balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
		}
	} else if balance.SentAssetAllocatedSat != nil {
		sentAssetAllocatedSat := balance.SentAssetAllocatedSat[assetGuid]
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[assetGuid]
		if ok {	
			balanceAssetAllocatedSat.Add(&balanceAssetAllocatedSat, totalAssetSentValue)
			sentAssetAllocatedSat.Sub(&sentAssetAllocatedSat, totalAssetSentValue)
			if sentAssetAllocatedSat.Sign() < 0 {
				d.resetValueSatToZero(&sentAssetAllocatedSat, assetSenderAddrDesc, "balance")
			}
			balance.SentAssetAllocatedSat[assetGuid] = sentAssetAllocatedSat
			balance.BalanceAssetAllocatedSat[assetGuid] = balanceAssetAllocatedSat
		} else {
			glog.Warningf("DisconnectAssetAllocationInput: Asset Allocated balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
		}
	} else {
		glog.Warningf("DisconnectAssetAllocationInput: Asset Sent Allocated balance not found guid %s (%s)", assetGuid, assetStrSenderAddrDesc)
	}
	return nil

}
func (d *RocksDB) ConnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		return err
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.ConnectAssetAllocationOutput(sptData, balances, version, addresses, btxID, outputIndex)
	} else if d.chainParser.IsAssetTx(version) {
		return d.ConnectAssetOutput(sptData, balances, version, addresses, btxID, outputIndex)
	}
	return nil
}

func (d *RocksDB) DisconnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		return err
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.DisconnectAssetAllocationOutput(sptData, balances, version, addresses)
	} else if d.chainParser.IsAssetTx(version) {
		return d.DisconnectAssetOutput(sptData, balances, version, addresses)
	}
	return nil
}
