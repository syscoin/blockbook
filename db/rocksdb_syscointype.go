package db

import (
	
	"bytes"
	"encoding/hex"
	"time"
	"fmt"
	
	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/flier/gorocksdb"
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/syscoinwire/syscoin/wire"
	vlq "github.com/bsm/go-vlq"
)
var AssetCache map[uint64]bchain.Asset
var SetupAssetCacheFirstTime bool = true
// GetTxAssetsCallback is called by GetTransactions/GetTxAssets for each found tx
type GetTxAssetsCallback func(txids []string) error

func (d *RocksDB) ConnectAllocationInput(addrDesc* bchain.AddressDescriptor, height uint32, version int32, balanceAsset *bchain.AssetBalance, btxID []byte, assetInfo* bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, txAssets bchain.TxAssetMap) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		return err
	}
	if dBAsset == nil {
		return errors.New(fmt.Sprint("ConnectAllocationInput could not read asset " , assetInfo.AssetGuid))
	}
	counted := d.addToAssetsMap(txAssets, assetInfo.AssetGuid, btxID, version, height)
	if !counted {
		dBAsset.Transactions++
		assets[assetInfo.AssetGuid] = dBAsset
	}
	counted = d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers++
	}
	balanceAsset.BalanceSat.Sub(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.BalanceSat.Sign() < 0 {
		balanceAsset.BalanceSat.SetInt64(0)
	}
	balanceAsset.SentSat.Add(balanceAsset.SentSat, assetInfo.ValueSat)
	return nil
}

func (d *RocksDB) ConnectAllocationOutput(addrDesc* bchain.AddressDescriptor, height uint32, balanceAsset *bchain.AssetBalance, version int32, btxID []byte, assetInfo* bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, txAssets bchain.TxAssetMap, memo []byte) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		dBAsset = &bchain.Asset{Transactions: 0, AssetObj: wire.AssetType{}, MetaData: memo}
	}
	counted := d.addToAssetsMap(txAssets, assetInfo.AssetGuid, btxID, version, height)
	if !counted {
		dBAsset.Transactions++
		assets[assetInfo.AssetGuid] = dBAsset
	}
	// asset guid + txid + address of output/input must match for counted to be true
	counted = d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers++
	}
	balanceAsset.BalanceSat.Add(balanceAsset.BalanceSat, assetInfo.ValueSat)
	return nil
}

func (d *RocksDB) DisconnectAllocationOutput(addrDesc *bchain.AddressDescriptor, balanceAsset *bchain.AssetBalance,  btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, assetFoundInTx func(asset uint64, btxID []byte) bool) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if dBAsset == nil || err != nil {
		if dBAsset == nil {
			return errors.New(fmt.Sprint("DisconnectAllocationOutput could not read asset " , assetInfo.AssetGuid))
		}
		return err
	}
	balanceAsset.BalanceSat.Sub(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.BalanceSat.Sign() < 0 {
		balanceAsset.BalanceSat.SetInt64(0)
	}
	exists := assetFoundInTx(assetInfo.AssetGuid, btxID)
	if !exists {
		dBAsset.Transactions--
	}
	counted := d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers--
	}
	assets[assetInfo.AssetGuid] = dBAsset
	return nil
}
func (d *RocksDB) DisconnectAllocationInput(addrDesc *bchain.AddressDescriptor, balanceAsset *bchain.AssetBalance,  btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, assetFoundInTx func(asset uint64, btxID []byte) bool) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if dBAsset == nil || err != nil {
		if dBAsset == nil {
			return errors.New(fmt.Sprint("DisconnectAllocationInput could not read asset " , assetInfo.AssetGuid))
		}
		return err
	}
	balanceAsset.SentSat.Sub(balanceAsset.SentSat, assetInfo.ValueSat)
	balanceAsset.BalanceSat.Add(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.SentSat.Sign() < 0 {
		balanceAsset.SentSat.SetInt64(0)
	}
	exists := assetFoundInTx(assetInfo.AssetGuid, btxID)
	if !exists {
		dBAsset.Transactions--
	}
	counted := d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers--
	}
	assets[assetInfo.AssetGuid] = dBAsset
	return nil
}

func (d *RocksDB) SetupAssetCache() error {
	start := time.Now()
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
	}
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetFillCache(false)
	it := d.db.NewIteratorCF(d.ro, d.cfh[cfAssets])
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		assetKey, _ := d.chainParser.UnpackVaruint64(it.Key().Data())
		assetDb, err := d.chainParser.UnpackAsset(it.Value().Data())
		if err != nil {
			return err
		}
		AssetCache[assetKey] = *assetDb
	}
	glog.Info("SetupAssetCache, ", time.Since(start))
	return nil
}


func (d *RocksDB) storeAssets(wb *gorocksdb.WriteBatch, assets map[uint64]*bchain.Asset) error {
	if assets == nil {
		return nil
	}
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
	}
	for guid, asset := range assets {
		AssetCache[guid] = *asset
		key := make([]byte, vlq.MaxLen64)
		l := d.chainParser.PackVaruint64(guid, key)
		key = key[:l]
		// total supply of -1 signals asset to be removed from db - happens on disconnect of new asset
		if asset.AssetObj.TotalSupply == -1 {
			delete(AssetCache, guid)
			wb.DeleteCF(d.cfh[cfAssets], key)
		} else {
			buf, err := d.chainParser.PackAsset(asset)
			if err != nil {
				return err
			}
			wb.PutCF(d.cfh[cfAssets], key, buf)
		}
	}
	return nil
}

func (d *RocksDB) GetAssetCache() *map[uint64]bchain.Asset {
	return &AssetCache
}

func (d *RocksDB) GetSetupAssetCacheFirstTime() bool {
	return SetupAssetCacheFirstTime
}

func (d *RocksDB) SetSetupAssetCacheFirstTime(cacheVal bool) {
	SetupAssetCacheFirstTime = cacheVal
}

func (d *RocksDB) GetBaseAssetID(guid uint64) uint64 {
	return guid & 0xffffffff
}
func (d *RocksDB) GetNFTID(guid uint64) uint64 {
	return guid >> 32
}

func (d *RocksDB) GetAsset(guid uint64, assets map[uint64]*bchain.Asset) (*bchain.Asset, error) {
	var assetDb *bchain.Asset
	var assetL1 *bchain.Asset
	var ok bool
	if assets != nil {
		if assetL1, ok = assets[guid]; ok {
			return assetL1, nil
		}
	}
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
		// so it will store later in cache
		ok = false
	} else {
		var assetDbCache, ok = AssetCache[guid]
		if ok {
			return &assetDbCache, nil
		}
	}
	key := make([]byte, vlq.MaxLen64)
	l := d.chainParser.PackVaruint64(guid, key)
	key = key[:l]
	val, err := d.db.GetCF(d.ro, d.cfh[cfAssets], key)
	if err != nil {
		return nil, err
	}
	// nil data means the key was not found in DB
	if val.Data() == nil {
		return nil, nil
	}
	defer val.Free()
	buf := val.Data()
	if len(buf) == 0 {
		return nil, errors.New("GetAsset: empty value in asset db")
	}
	assetDb, err = d.chainParser.UnpackAsset(buf)
	if err != nil {
		return nil, err
	}
	// cache miss, add it, we also add it on storeAsset but on API queries we should not have to wait until a block
	// with this asset to store it in cache
	if !ok {
		AssetCache[guid] = *assetDb
	}
	return assetDb, nil
}

func (d *RocksDB) storeTxAssets(wb *gorocksdb.WriteBatch, txassets bchain.TxAssetMap) error {
	for key, txAsset := range txassets {
		buf := d.chainParser.PackAssetTxIndex(txAsset)
		wb.PutCF(d.cfh[cfTxAssets], []byte(key), buf)
	}
	return nil
}

// GetTxAssets finds all asset transactions for each asset
// Transaction are passed to callback function in the order from newest block to the oldest
func (d *RocksDB) GetTxAssets(assetGuid uint64, lower uint32, higher uint32, assetsBitMask bchain.AssetsMask, fn GetTxAssetsCallback) (err error) {
	startKey := d.chainParser.PackAssetKey(assetGuid, higher)
	stopKey := d.chainParser.PackAssetKey(assetGuid, lower)
	it := d.db.NewIteratorCF(d.ro, d.cfh[cfTxAssets])
	defer it.Close()
	for it.Seek(startKey); it.Valid(); it.Next() {
		key := it.Key().Data()
		val := it.Value().Data()
		if bytes.Compare(key, stopKey) > 0 {
			break
		}
		txIndexes := d.chainParser.UnpackAssetTxIndex(val)
		if txIndexes != nil {
			txids := []string{}
			for _, txIndex := range txIndexes {
				mask := uint32(txIndex.Type)
				if (assetsBitMask == bchain.AllMask) || ((uint32(assetsBitMask) & mask) == mask) {
					txids = append(txids, hex.EncodeToString(txIndex.BtxID))
				}
			}
			if len(txids) > 0 {
				if err := fn(txids); err != nil {
					if _, ok := err.(*StopIteration); ok {
						return nil
					}
					return err
				}
			}
		}
	}
	return nil
}

// addToAssetsMap maintains mapping between assets and transactions in one block
// the return value is true if the tx was processed before, to not to count the tx multiple times
func (d *RocksDB) addToAssetsMap(txassets bchain.TxAssetMap, assetGuid uint64, btxID []byte, version int32, height uint32) bool {
	// check that the asset was already processed in this block
	// if not found, it has certainly not been counted
	key := string(d.chainParser.PackAssetKey(assetGuid, height))
	at, found := txassets[key]
	if found {
		// if the tx is already in the slice
		for _, t := range at.Txs {
			if bytes.Equal(btxID, t.BtxID) {
				return true
			}
		}
	} else {
		at = &bchain.TxAsset{Txs: []*bchain.TxAssetIndex{}}
		txassets[key] = at
	}
	at.Txs = append(at.Txs, &bchain.TxAssetIndex{Type: d.chainParser.GetAssetsMaskFromVersion(version), BtxID: btxID})
	at.Height = height
	return false
}
// to control Transfer add/remove
func (d *RocksDB) addToAssetAddressMap(txassetAddresses bchain.TxAssetAddressMap, assetGuid uint64, btxID []byte, addrDesc *bchain.AddressDescriptor) bool {
	at, found := txassetAddresses[assetGuid]
	if found {
		// if the tx is already in the slice
		for _, t := range at.Txs {
			if bytes.Equal(btxID, t.BtxID) && bytes.Equal(*addrDesc, t.AddrDesc) {
				return true
			}
		}
	} else {
		at = &bchain.TxAssetAddress{Txs: []*bchain.TxAssetAddressIndex{}}
		txassetAddresses[assetGuid] = at
	}
	at.Txs = append(at.Txs, &bchain.TxAssetAddressIndex{AddrDesc: *addrDesc, BtxID: btxID})
	return false
}