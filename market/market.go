package market

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"math"
	"net/http"
	"sync"
)

func mapKeyExist(key int32, value map[int32]*MData) (bool, any) {
	if val, ok := value[key]; ok {
		return true, val
	} else {
		return false, nil
	}
}

// MDataMapLocker MDataMap with sync.RWMutex locker
type MDataMapLocker struct {
	lock sync.Mutex
	Map  MDataMap
}

func (m *MDataMapLocker) Dumper() (mdm MDataMap) {
	m.lock.Lock()
	mdm = m.Map
	m.lock.Unlock()
	return
}

func (m *MDataMapLocker) Loader(prc MDataMap) (*MDataMapLocker, error) {
	m.loader(prc)
	return m, nil
}

func (m *MDataMapLocker) loader(prc MDataMap) {
	for i := range prc {
		m.lock.Lock()
		if ok, _ := mapKeyExist(i, m.Map); ok {
			j := m.Map[i]
			if (j.Sell.Max < prc[i].Sell.Max || math.Abs(j.Sell.Max) <= 0.001) && prc[i].Sell.Max >= 0.01 {
				j.Sell.Max = prc[i].Sell.Max
			}
			if (j.Sell.Min > prc[i].Sell.Min || math.Abs(j.Sell.Min) <= 0.001) && prc[i].Sell.Min >= 0.01 {
				j.Sell.Min = prc[i].Sell.Min
			}
			if (j.Buy.Max < prc[i].Buy.Max || math.Abs(j.Buy.Max) <= 0.001) && prc[i].Buy.Max >= 0.01 {
				j.Buy.Max = prc[i].Buy.Max
			}
			if (j.Buy.Min > prc[i].Buy.Min || math.Abs(j.Buy.Min) <= 0.001) && prc[i].Buy.Min >= 0.01 {
				j.Buy.Min = prc[i].Buy.Min
			}
			m.Map[i] = j
		} else {
			m.Map[i] = prc[i]
		}
		m.lock.Unlock()
	}
}

// MDataMap a map of MData
type MDataMap map[int32]*MData

func (m MDataMap) DatabaseUpdate(server string) (err error) {
	const SqlSerenity string = "update `serenity` set buy_max=?, sell_max=?, buy_min=?, sell_min=? where id=?"
	const SqlTranquility string = "update `tranquility` set buy_max=?, sell_max=?, buy_min=?, sell_min=? where id=?"
	var sqlStmt string
	if server == "serenity" {
		sqlStmt = SqlSerenity
	} else if server == "tranquility" {
		sqlStmt = SqlTranquility
	} else {
		return errors.New(fmt.Sprintf("Invalid server %s", server))
	}
	Db, err := sql.Open("mysql", "root:yaoyao321wang@tcp(localhost:3306)/evemarket")
	if err != nil {
		return
	}
	stmt, err := Db.Prepare(sqlStmt)
	if err != nil {
		return
	}
	for i := range m {
		_, err = stmt.Exec(m[i].Buy.Max, m[i].Sell.Max, m[i].Buy.Min, m[i].Sell.Min, m[i].TypeId)
		if err != nil {
			return
		}
	}
	return nil
}

// MData the primary price handler
type MData struct {
	TypeId int32 `json:"type_id"`
	Sell   struct {
		Max float64 `json:"max"`
		Min float64 `json:"min"`
	} `json:"sell"`
	Buy struct {
		Max float64 `json:"max"`
		Min float64 `json:"min"`
	} `json:"buy"`
}

type PriceData struct {
	Duration     int32   `json:"duration"`
	IsBuyOrder   bool    `json:"is_buy_order"`
	Issued       string  `json:"issued"`
	LocationId   int64   `json:"location_id"`
	MinVolume    int32   `json:"min_volume"`
	OrderId      int64   `json:"order_id"`
	Price        float64 `json:"price"`
	Range        string  `json:"range"`
	SystemId     int32   `json:"system_id"`
	TypeId       int32   `json:"type_id"`
	VolumeRemain int32   `json:"volume_remain"`
	VolumeTotal  int32   `json:"volume_total"`
}

type PriceDataCollection []PriceData

func (s PriceDataCollection) Sort() (m MDataMap) {
	var v int32
	m = MDataMap{}
	for i := range s {
		v = s[i].TypeId
		prc := s[i].Price
		if ok, _ := mapKeyExist(s[i].TypeId, m); ok {
			if s[i].IsBuyOrder {
				if m[v].Buy.Max < prc || m[v].Buy.Max == 0.0 {
					m[v].Buy.Max = prc
				} else if m[v].Buy.Min > prc || m[v].Buy.Min == 0.0 {
					m[v].Buy.Min = prc
				}
			} else {
				if m[v].Sell.Max < prc || m[v].Sell.Max == 0.0 {
					m[v].Sell.Max = prc
				} else if m[v].Sell.Min > prc || m[v].Sell.Min == 0.0 {
					m[v].Sell.Min = prc
					//if v == 34 {
					//	fmt.Println(prc, m[v], "smaller")
					//}
				}
			}
		} else {
			m[v] = &MData{TypeId: s[i].TypeId}
			if s[i].IsBuyOrder {
				m[v].Buy.Max = prc
				m[v].Buy.Min = prc
				m[v].Sell = struct {
					Max float64 `json:"max"`
					Min float64 `json:"min"`
				}{Max: 0.0, Min: 0.0}
			} else {
				m[v].Sell.Max = prc
				m[v].Sell.Min = prc
				m[v].Buy = struct {
					Max float64 `json:"max"`
					Min float64 `json:"min"`
				}{Max: 0.0, Min: 0.0}
			}
		}
	}
	return
}

type eBoardCast struct {
	lock sync.RWMutex
	page int32
}

func mktRequest(url string, d *MDataMapLocker, client *http.Client) (*MDataMapLocker, error, bool) {
	//defer func() {
	//	defer func() {
	//		recover()
	//	}()
	//	if d.Map[34].Sell.Min <= 1.0 {
	//		fmt.Println(url)
	//	}
	//}()
	data, err := client.Get(url)
	if err != nil {
		return d, err, false
	}
	if data.StatusCode == 404 {
		return d, nil, true
	}
	decoder := json.NewDecoder(data.Body)
	dt := make(PriceDataCollection, 1)
	err = decoder.Decode(&dt)
	if err != nil {
		return d, err, false
	}
	vData := dt.Sort()
	d, err = d.Loader(vData)
	if err != nil {
		return d, err, false
	}
	return d, nil, false
}

func reqUrlBuffer(server string, regionId int32, page int32) (u string, err error) {
	SerenityUrlBase := "https://esi.evepc.163.com/latest/markets/%d/orders/?datasource=serenity&order_type=all&page=%d"
	TranquilityUrlBase := "https://esi.evetech.net/latest/markets/%d/orders/?datasource=tranquility&order_type=all&page=%d"
	if server == "serenity" {
		return fmt.Sprintf(SerenityUrlBase, regionId, page), nil
	} else if server == "tranquility" {
		return fmt.Sprintf(TranquilityUrlBase, regionId, page), nil
	} else {
		return "", errors.New(fmt.Sprintf("invalid server %s", server))
	}
}

func marketRequestSender(d *MDataMapLocker, server string, regionId int32, page int32, client *http.Client) (err error, status bool) {
	url, err := reqUrlBuffer(server, regionId, page)
	if err != nil {
		status = true
		return
	}
	_, err, finished := mktRequest(url, d, client)
	if err != nil {
		status = true
		return
	}
	if finished {
		status = false
		return
	}
	return nil, true
}

func marketRequestHandler(d *MDataMapLocker, server string, regionId int32, page int32, client *http.Client, eBd *eBoardCast, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	eBd.lock.RLock()
	ePage := eBd.page
	eBd.lock.RUnlock()
	if page > ePage && ePage != 0 {
		//fmt.Println(page)
		return
	}
Req:
	err, status := marketRequestSender(d, server, regionId, page, client)
	if err != nil {
		fmt.Printf("第 %d 页 出错 - 回滚\n", page)
		goto Req
	}
	if !status {
		eBd.lock.Lock()
		eBd.page = page
		eBd.lock.Unlock()
		//fmt.Println(page)
		return
	}
	return
}

func MktRequestsDistributor(server string, regionId int32) (m MDataMap, err error) {
	var rg int32
	wg := &sync.WaitGroup{}
	if server == "serenity" {
		rg = 120
	} else if server == "tranquility" {
		rg = 300
	} else {
		return nil, errors.New(fmt.Sprintf("invalid server %s", server))
	}
	md := MDataMapLocker{
		Map: MDataMap{},
	}
	client := http.Client{}
	eChan := eBoardCast{page: 0}
	var i int32
	for i = 1; i < rg; i++ {
		wg.Add(1)
		go marketRequestHandler(&md, server, regionId, i, &client, &eChan, wg)
		//fmt.Println(md.Map[34], i)
		if i%25 == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
	for ; i < rg+150; i++ {
		wg.Add(1)
		go marketRequestHandler(&md, server, regionId, i, &client, &eChan, wg)
		//fmt.Println(md.Map[34], i)
		wg.Wait()
	}
	wg.Wait()
	md.lock.Lock()
	m = md.Map
	md.lock.Unlock()
	return m, nil
}
