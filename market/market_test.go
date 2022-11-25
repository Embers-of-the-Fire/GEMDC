package market

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestMarketRequestsDistributor(t *testing.T) {
	v, err := MktRequestsDistributor("serenity", 10000002)
	jv, err := json.Marshal(v)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	njf, err := os.OpenFile("test.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	_, err = njf.Write(jv)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	fmt.Println("Write Finished")
	err = v.DatabaseUpdate("serenity")
	if err != nil {
		return
	}
	fmt.Println("Db Updated")
}
