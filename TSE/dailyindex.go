package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const urlTSEIndexValue = "http://www.twse.com.tw/indicesReport/MI_5MINS_HIST?response=json&date=%4d%02d%02d"
const urlTSEIndexTrade = "http://www.twse.com.tw/exchangeReport/FMTQIK?response=json&date=%4d%02d%02d"
const urlTSEIndexInvestor = "http://www.twse.com.tw/fund/BFI82U?response=json&dayDate=%4d%02d%02d&type=day"
const urlTSEIndexMarginShort = "http://www.twse.com.tw/exchangeReport/MI_MARGN?response=json&date=%4d%02d%02d&selectType=MS"
const kMinSize = 1024
const kMinDate = 20000000

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

const indexCose = "TAIEX"

type IndexValue struct {
	Date string
	Data [][]string `json:"data"`
}

type IndexTrade struct {
	Date  string
	Trade [][]string `json:"data"`
}

type IndexInvestor struct {
	Date     string
	Investor [][]string `json:"data"`
}

type IndexMarginShort struct {
	Date   string
	Margin [][]string `json:"creditList"`
}

var flagFromDate = flag.Int("f", 0, "from date YYYYMMDD (default: the day after latest trade date in DB)")
var flagToDate = flag.Int("t", 0, "to date YYYYMMDD (default: today)")
var flagLastTradeDay = flag.Bool("l", false, "last trade day only")

func main() {
	flag.Parse()
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", DB_USER, DB_PASSWORD, DB_NAME, DB_HOST)
	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Println(err)
		return
	}

	today := time.Now()
	fromDate := *flagFromDate
	toDate := *flagToDate
	if fromDate == 0 || fromDate < kMinDate {
		fromDate, err = fetchLastTradeDate(db)
		if err != nil {
			fromDate = int(today.Year())*10000 + int(today.Month())*100 + today.Day()
		}
	}
	if toDate == 0 || toDate < kMinDate {
		toDate = today.Year()*10000 + int(today.Month())*100 + today.Day()
	}

	log.Println(fromDate, toDate)

	local, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		panic(err)
	}

	var quoteDate, beginDate time.Time
	if *flagLastTradeDay {
		quoteDate = today
	} else {
		quoteDate = time.Date(toDate/10000, time.Month(toDate%10000/100), toDate%100, 1, 0, 0, 0, local)
	}
	beginDate = time.Date(fromDate/10000, time.Month(fromDate%10000/100), fromDate%100, 0, 0, 0, 0, local)
	log.Println("beginDate = ", beginDate)
	for quoteDate.After(beginDate) {
		log.Println(quoteDate)
		o, h, l, c, ok := fetchIndexValue(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if !ok {
			break
		}
		fmt.Println(o, h, l, c)
		if *flagLastTradeDay {
			break
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func fetchLastTradeDate(db *sql.DB) (int, error) {
	var row1 string
	sqlString := "SELECT to_char(MAX(trade_date)+interval '1 day', 'YYYYMMDD') FROM index_values"
	err := db.QueryRow(sqlString).Scan(&row1)
	if err != nil {
		return 0, err
	}
	log.Println("the day after latest trade day = ", row1)

	return strconv.Atoi(row1)
}

func readIndexCode(db *sql.DB) map[string]string {
	sqlString := "SELECT * FROM indices;"
	rows, err := db.Query(sqlString)
	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	codes := make(map[string]string)
	for rows.Next() {
		var code string
		var name string
		if err := rows.Scan(&code, &name); err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("code %s name is %s\n", code, name)
		codes[name] = code
	}

	return codes
}

func fetchIndexValue(year int, month int, day int) (float64, float64, float64, float64, bool) {
	var url string
	var contents []byte
	var o, h, l, c float64

	url = fmt.Sprintf(urlTSEIndexValue, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return o, h, l, c, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return o, h, l, c, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return o, h, l, c, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	if len(contents) < kMinSize {
		return o, h, l, c, false
	}

	var indexValue IndexValue
	err = json.Unmarshal(contents, &indexValue)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return o, h, l, c, false
	}

	dateTW := fmt.Sprintf("%3d/%02d/%02d", year-1911, month, day)
	fmt.Println(dateTW)
	for _, data := range indexValue.Data {
		for i := 0; i < len(data); i++ {
			if data[0] == dateTW {
				fmt.Println(data)
				v := strings.Replace(data[1], ",", "", -1)
				if o, err := strconv.ParseFloat(v, 64); err == nil {
					return o, h, l, c, false
				}
				v = strings.Replace(data[2], ",", "", -1)
				if h, err := strconv.ParseFloat(v, 64); err == nil {
					return o, h, l, c, false
				}
				v = strings.Replace(data[3], ",", "", -1)
				if l, err := strconv.ParseFloat(v, 64); err == nil {
					return o, h, l, c, false
				}
				v = strings.Replace(data[4], ",", "", -1)
				if c, err := strconv.ParseFloat(v, 64); err == nil {
					return o, h, l, c, false
				}
			}
		}
	}

	return o, h, l, c, true
}
