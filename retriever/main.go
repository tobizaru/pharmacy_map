// MIT License

// Copyright (c) 2022 tobizaru

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

//薬局一覧EXCELの列及びその意味
const (
	PREFID  = 1  //都道府県コード
	PREF    = 2  //都道府県名
	CLASS   = 3  //区分
	ID      = 4  //医療機関番号
	NAME    = 7  //医療機関名称
	POSTID  = 8  //医療機関所在地（郵便番号）
	ADDRESS = 9  //医療機関所在地（住所）
	TEL     = 10 //電話番号
	FAX     = 11 //FAX番号
	FACIL   = 13 //受理届出名称
)

const (
	//EXCELのURL一覧
	EXCEL_JSON = "xls_urls.yml"
	//受理届出名称と調剤報酬点数
	REWARD_JSON = "reward.yml"
	//出力
	OUTPUT_JSON = "pharmacy.json"
)

//EXCELのURL一覧JSONの構造
type ExcelInfo struct {
	Department string   `yaml:"department"` //管轄局
	OriginURL  string   `yaml:"originURL"`  //ExcelへのリンクのあるURL
	RewardID   string   `yaml:"reward_id"`  //診療報酬ID
	ExcelURL   []string `yaml:"excel_url"`  //EXCELのURL
	Desc       string   `yaml:"desc"`       //注意書き
}

//薬局情報出力のJSON構造
type PharmacyInfo struct {
	PrefectureID string   `json:"prefecture_id"` //都道府県コード
	Prefecture   string   `json:"prefecture"`    //都道府県名
	ID           string   `json:"id"`            //医療機関番号
	Name         string   `json:"name"`          //医療機関名称
	PostID       string   `json:"post_id"`       //医療機関所在地（郵便番号）
	Address      string   `json:"address"`       //医療機関所在地（住所）
	Telephone    string   `json:"telephone"`     //電話番号
	Fax          string   `json:"fax"`           //FAX番号
	Facility     []string `json:"facility"`      //受理届出名称

	RewardID string  `yaml:"reward_id"`      //診療報酬ID
	Point    int     `json:"point"`          //調剤報酬点数
	Lat      float64 `json:"lat"`            //緯度
	Lon      float64 `json:"lon"`            //経度
	Desc     string  `json:"desc,omitempty"` //注意書き
}

//住所→緯度・経度変換URL
const GEOCODE_URL = "https://geocode.csis.u-tokyo.ac.jp/cgi-bin/simple_geocode.cgi"

//geocodeからの結果
type GeocodeResult struct {
	Candidate []struct {
		Longitude float64 `xml:"longitude"` //経度
		Latitude  float64 `xml:"latitude"`  //緯度
	} `xml:"candidate"`
}

//調剤報酬JSONの構造
type RewardInfo struct {
	ID     string         `yaml:"id"`
	Reward map[string]int `yaml:"reward"`
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	//EXCELのURL一覧JSONからURL抽出
	excels, err := readExcelInfo()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	//薬局EXCELファイル読み込みと薬局情報取得
	pharmacy, err := retrieveExcel(excels)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	//追加の薬局情報（緯度経度、調剤報酬）取得
	if err = retrieveAdditionalInfo(pharmacy); err != nil {
		log.Fatalf("%+v", err)
	}

	//薬局情報出力のJSON出力
	if err = writeJSON(pharmacy); err != nil {
		log.Fatalf("%+v", err)
	}
}

//EXCELのURL一覧JSONからURL抽出
func readExcelInfo() ([]*ExcelInfo, error) {
	var excels []*ExcelInfo
	raw, err := ioutil.ReadFile(EXCEL_JSON)
	if err != nil {
		return nil, errors.New(EXCEL_JSON + " not found")
	}
	if err = yaml.Unmarshal(raw, &excels); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal param")
	}
	return excels, nil
}

//薬局EXCELファイル読み込みと薬局情報取得
func retrieveExcel(excels []*ExcelInfo) ([]*PharmacyInfo, error) {
	log.Println("EXCEL及び薬局情報を取得中...")
	var pharmacy []*PharmacyInfo
	for _, infos := range excels {
		for _, info := range infos.ExcelURL {
			//各EXCEL　URLをHTTP GET
			log.Printf("  URL: %s", info)
			u, err := url.Parse(info)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse excel url")
			}
			resp, err := http.Get(u.String())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get excel")
			}
			defer resp.Body.Close()
			switch path.Ext(u.Path) {
			case ".xlsx":
				//取得したファイルがEXCELならそのEXCEL読み込み薬局情報取得
				p, err := extractPharmacy(resp.Body, infos)
				if err != nil {
					return nil, errors.Wrap(err, "failed to extrahct pharmacy info")
				}
				log.Printf("    %d 個の薬局情報取得済み", len(p))
				pharmacy = append(pharmacy, p...)
			case ".zip":
				//取得したファイルがZIPなら
				bs, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return nil, errors.Wrap(err, "failed to read resp body")
				}
				//zipファイルを読み込み
				buf := bytes.NewReader(bs)
				r, err := zip.NewReader(buf, int64(buf.Size()))
				if err != nil {
					return nil, errors.Wrap(err, "failed to create a reader")
				}
				for _, f := range r.File {
					//中のファイルがEXCELファイルなら、読み込み薬局情報取得
					if path.Ext(f.Name) != ".xlsx" {
						continue
					}
					fname, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), f.Name)
					if err != nil {
						return nil, errors.Wrap(err, "failed to transform the file name")
					}
					log.Printf("    zip内ファイル: %s", fname)
					rc, err := f.Open()
					if err != nil {
						return nil, errors.Wrap(err, "failed to open a file in zip")
					}
					defer rc.Close()
					p, err := extractPharmacy(rc, infos)
					if err != nil {
						return nil, errors.Wrap(err, "failed to extrahct pharmacy info")
					}
					log.Printf("      %d 個の薬局情報取得済み", len(p))
					pharmacy = append(pharmacy, p...)
				}
			default:
				return nil, errors.Errorf("unknown extention %s", path.Ext(info))
			}
		}
	}
	log.Printf("合計 %d 個の薬局情報取得済み", len(pharmacy))
	return pharmacy, nil
}

//EXCELファイルを読み込み薬局情報取得
func extractPharmacy(r io.Reader, infos *ExcelInfo) ([]*PharmacyInfo, error) {
	//EXCEL読み込み
	e, err := excelize.OpenReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open an excel")
	}
	defer e.Close()
	//シートの列データ取得
	rows, err := e.GetRows("Sheet1")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rows")
	}

	ps := make([]*PharmacyInfo, 0, len(rows))
	var p *PharmacyInfo
	for j, row := range rows {
		//まずセル内のの前後スペース除去
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		//各行の列数を一定以下なら空データを追加
		//（後にアクセスしやすいように）
		for i := len(row); i <= FACIL; i++ {
			rows[j] = append(rows[j], "")
		}
	}
	//ヘッダ行数取得
	i := 0
	for i = 0; i < len(rows); i++ {
		if len(rows[i]) > ID && rows[i][ID] == "医療機関番号" {
			break
		}
	}
	//最後まで言ってしまったらエラー
	if i == len(rows) {
		return nil, errors.New("cannto find the start line in excel")
	}
	for _, row := range rows[i+1:] {
		//薬局以外は無視
		if row[CLASS] != "薬局" {
			continue
		}
		//各列でIDが変わったら別の薬局、新規薬局データを作成
		if p == nil || p.ID != row[ID] {
			p = &PharmacyInfo{
				PrefectureID: row[PREFID],
				Prefecture:   row[PREF],
				ID:           row[ID],
				Name:         row[NAME],
				PostID:       row[POSTID],
				Address:      row[ADDRESS],
				Telephone:    row[TEL],
				Fax:          row[FAX],
				Desc:         infos.Desc,
				RewardID:     infos.RewardID,
			}
			ps = append(ps, p)
		}
		//受理届出名称を追加
		if row[FACIL] != "" {
			p.Facility = append(p.Facility, row[FACIL])
		}
	}
	return ps, nil
}

//追加の薬局情報（緯度経度、調剤報酬）取得
func retrieveAdditionalInfo(infos []*PharmacyInfo) error {
	const RETRY = 10

	log.Println("薬局の緯度経度及び調剤報酬加算を取得中...")
	//調剤報酬情報をJSONファイルから取得
	var rewardInfo []*RewardInfo
	raw, err := ioutil.ReadFile(REWARD_JSON)
	if err != nil {
		return errors.New(REWARD_JSON + " not found")
	}
	if err = yaml.Unmarshal(raw, &rewardInfo); err != nil {
		return errors.Wrap(err, "failed to unmarshal point info")
	}

	searchURL, err := url.Parse(GEOCODE_URL)
	if err != nil {
		return errors.Wrap(err, "failed to parse search url")
	}
	for i, info := range infos {
		if i%100 == 0 || i == len(infos)-1 {
			log.Printf("  %d/%d個目を処理中", i, len(infos))
		}

		//ジオコーディングURLから緯度経度取得
		//クエリとして住所追加
		q := searchURL.Query()
		q.Set("addr", info.Address)
		searchURL.RawQuery = q.Encode()
		var result *GeocodeResult
		for i := 0; i < RETRY; i++ {
			//RETRYは10回まで、RETRYごとにウエイト
			time.Sleep(10 * time.Second * time.Duration(i))
			var resp *http.Response
			resp, err = http.Get(searchURL.String())
			if err != nil {
				log.Printf("failed to get address , retry %d, %+v", i, err)
				continue
			}
			defer resp.Body.Close()
			if err = xml.NewDecoder(resp.Body).Decode(&result); err != nil {
				log.Printf("failed to decode address , retry %d, %+v", i, err)
				continue
			}
			if len(result.Candidate) == 0 {
				log.Printf("failed to get result , retry %d, %+v", i, err)
				err = errors.New("invalid coordinate")
				continue
			}
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get address search")
		}
		//緯度経度情報を設定
		info.Lon = result.Candidate[0].Longitude
		info.Lat = result.Candidate[0].Latitude

		//この薬局の診療報酬データを探す
		var rinfo *RewardInfo
		for _, rinfo = range rewardInfo {
			if rinfo.ID == info.RewardID {
				break
			}
		}
		if rinfo == nil {
			return errors.New("failed to find reward info")
		}
		//指定された調剤報酬があれば報酬追加
		for _, f := range info.Facility {
			if p, ok := rinfo.Reward[f]; ok {
				info.Point += p
			}
		}
	}
	return nil
}

//薬局情報出力のJSON出力
func writeJSON(infos []*PharmacyInfo) error {
	w, err := os.Create(OUTPUT_JSON)
	if err != nil {
		return errors.Wrap(err, "failed to create a json file")
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err = enc.Encode(infos); err != nil {
		return errors.Wrap(err, "failed to encode json")
	}
	return nil
}
