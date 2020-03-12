package proc

import (
	"encoding/json"
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

func Populate(r *obj.Repo) error {
	fmt.Println("выгрузка данных из БД в Index файл")
	//type pack struct {
	//   Hash  string            `json:"phash"`
	//   Alias string            `json:"alias"`
	//   Files map[string]string `json:"files"`
	//}

	type packages map[string]obj.HashedPackData

	//p1 := pack{
	//    Hash:  "ed4174103c5cae3016b0b7bdeeb41c6bbf43897e",
	//    Alias: "",
	//    Files: map[string]string{
	//        "file_01": "1cadb18d682e1b45da91cb92d0d906ef6b552775",
	//        "file_02": "b8cc25e1e809325e9aa620007a9d9155f8303023",
	//    },
	//}
	//p2 := pack{
	//    Hash:  "61f581282d4803c60aaf3bf5a8d756694d1972f6",
	//    Alias: "",
	//    Files: map[string]string{
	//        "file_01": "8227495cbead4012f2f87cf8317fbd999fce2a1e",
	//        "file_02": "fbff681187dc61e91ccda4c1836fecc750e71456",
	//    },
	//}
	//data := packages{
	//   "pack_1": p1,
	//   "pack_2": p2,
	//}
	//
	//
	//

	packCh := make(chan obj.HashedPackData)
	packDataList := packages{}

	go r.HashedPackages(packCh) // список пакетов в из БД

	for packData := range packCh {

		//fmt.Println(packData.Id, packData.Name, packData.Hash)

		//p := pack{
		//    Hash: packData.Hash,
		//    Alias: packData.Alias,
		//}
		//

		packDataList[packData.Name] = packData // добавляем в map
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fmt.Println(string(jsonData))

	return nil
}
