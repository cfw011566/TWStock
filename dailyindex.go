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

const (
	urlTSEIndexValue       = "http://www.twse.com.tw/indicesReport/MI_5MINS_HIST?response=json&date=%4d%02d%02d"
	urlTSEIndexTrade       = "http://www.twse.com.tw/exchangeReport/FMTQIK?response=json&date=%4d%02d%02d"
	urlTSEIndexInvestor    = "http://www.twse.com.tw/fund/BFI82U?response=json&dayDate=%4d%02d%02d&type=day"
	urlTSEIndexMarginShort = "http://www.twse.com.tw/exchangeReport/MI_MARGN?response=json&date=%4d%02d%02d&selectType=MS"
	kMinDate               = 20000000
)

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

const indexCose = "TAIEX"

type jsonContent struct {
	Status string `json:"stat"`
	Date   string
	Data   [][]string `json:"data"`
}

type jsonContent2 struct {
	Status string `json:"stat"`
	Date   string
	Data   [][]string `json:"creditList"`
}

type Investor struct {
	Buy        string
	Sell       string
	Difference string
}

type IndexInvestor struct {
	DealerSelf  Investor
	DealerHedge Investor
	Trust       Investor
	Foreign     Investor
	Total       Investor
}

type MarginShortFields struct {
	TodayNew    string
	Redemption  string
	Outstanding string
	LastRemain  string
	TodayRemain string
}

type IndexMarginShort struct {
	Margin      MarginShortFields
	Short       MarginShortFields
	MarginValue MarginShortFields
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
		fmt.Println("-------")
		fmt.Println(quoteDate)
		time.Sleep(100 * time.Millisecond)
		o, h, l, c, ok := fetchIndexValue(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if !ok {
			log.Println(o, h, l, c)
			quoteDate = quoteDate.AddDate(0, 0, -1)
			continue
		}
		time.Sleep(100 * time.Millisecond)
		volume, amount, count, ok := fetchIndexTrade(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if !ok {
			log.Println(volume, amount, count)
			quoteDate = quoteDate.AddDate(0, 0, -1)
			continue
		}
		time.Sleep(100 * time.Millisecond)
		indexInvestor, ok := fetchIndexInvestor(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if !ok {
			log.Println(indexInvestor)
			quoteDate = quoteDate.AddDate(0, 0, -1)
			continue
		}
		time.Sleep(100 * time.Millisecond)
		indexMarginShort, ok := fetchIndexMarginShort(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if !ok {
			log.Println(indexInvestor)
			quoteDate = quoteDate.AddDate(0, 0, -1)
			continue
		}

		fmt.Println(o, h, l, c)
		fmt.Println(volume, amount, count)
		fmt.Println(indexInvestor)
		fmt.Println(indexMarginShort)

		dateString := fmt.Sprintf("%4d%02d%02d", quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		writeIndexQuote(db, dateString, o, h, l, c, volume, amount, count)
		writeIndexInvestor(db, dateString, indexInvestor)
		writeIndexMarginShort(db, dateString, indexMarginShort)

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

func fetchIndexValue(year int, month int, day int) (string, string, string, string, bool) {
	var url string
	var contents []byte
	var o, h, l, c string
	var jsonData jsonContent

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

	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return o, h, l, c, false
	}

	if jsonData.Status != "OK" {
		log.Println("stat: ", jsonData.Status)
		return o, h, l, c, false
	}

	dateTW := fmt.Sprintf("%3d/%02d/%02d", year-1911, month, day)
	//fmt.Println(dateTW)
	for _, data := range jsonData.Data {
		for i := 0; i < len(data); i++ {
			if data[0] == dateTW {
				//fmt.Println(data)
				o = strings.Replace(data[1], ",", "", -1)
				h = strings.Replace(data[2], ",", "", -1)
				l = strings.Replace(data[3], ",", "", -1)
				c = strings.Replace(data[4], ",", "", -1)
				break
			}
		}
	}

	return o, h, l, c, true
}

func fetchIndexTrade(year int, month int, day int) (string, string, string, bool) {
	var url string
	var contents []byte
	var volume, amount, count string
	var jsonData jsonContent

	url = fmt.Sprintf(urlTSEIndexTrade, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return volume, amount, count, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return volume, amount, count, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return volume, amount, count, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return volume, amount, count, false
	}

	if jsonData.Status != "OK" {
		log.Println("stat: ", jsonData.Status)
		return volume, amount, count, false
	}

	dateTW := fmt.Sprintf("%3d/%02d/%02d", year-1911, month, day)
	//fmt.Println(dateTW)
	for _, data := range jsonData.Data {
		for i := 0; i < len(data); i++ {
			if data[0] == dateTW {
				//fmt.Println(data)
				volume = strings.Replace(data[1], ",", "", -1)
				amount = strings.Replace(data[2], ",", "", -1)
				count = strings.Replace(data[3], ",", "", -1)
				break
			}
		}
	}

	return volume, amount, count, true
}

func fetchIndexInvestor(year int, month int, day int) (IndexInvestor, bool) {
	var url string
	var contents []byte
	var jsonData jsonContent
	var indexInvestor IndexInvestor

	url = fmt.Sprintf(urlTSEIndexInvestor, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return indexInvestor, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return indexInvestor, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return indexInvestor, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return indexInvestor, false
	}

	if jsonData.Status != "OK" {
		log.Println("stat: ", jsonData.Status)
		return indexInvestor, false
	}

	for _, data := range jsonData.Data {
		buy := strings.Replace(data[1], ",", "", -1)
		sell := strings.Replace(data[2], ",", "", -1)
		difference := strings.Replace(data[3], ",", "", -1)
		if data[0] == "自營商(自行買賣)" {
			indexInvestor.DealerSelf.Buy = buy
			indexInvestor.DealerSelf.Sell = sell
			indexInvestor.DealerSelf.Difference = difference
		} else if data[0] == "自營商(避險)" {
			indexInvestor.DealerHedge.Buy = buy
			indexInvestor.DealerHedge.Sell = sell
			indexInvestor.DealerHedge.Difference = difference
		} else if data[0] == "投信" {
			indexInvestor.Trust.Buy = buy
			indexInvestor.Trust.Sell = sell
			indexInvestor.Trust.Difference = difference
		} else if data[0] == "外資及陸資" {
			indexInvestor.Foreign.Buy = buy
			indexInvestor.Foreign.Sell = sell
			indexInvestor.Foreign.Difference = difference
		} else if data[0] == "合計" {
			indexInvestor.Total.Buy = buy
			indexInvestor.Total.Sell = sell
			indexInvestor.Total.Difference = difference
		} else {
			break
		}
	}

	return indexInvestor, true
}

func fetchIndexMarginShort(year int, month int, day int) (IndexMarginShort, bool) {
	var url string
	var contents []byte
	var jsonData jsonContent2
	var marginShort IndexMarginShort

	url = fmt.Sprintf(urlTSEIndexMarginShort, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return marginShort, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return marginShort, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return marginShort, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return marginShort, false
	}

	if jsonData.Status != "OK" {
		log.Println("stat: ", jsonData.Status)
		return marginShort, false
	}

	for _, data := range jsonData.Data {
		todayNew := strings.Replace(data[1], ",", "", -1)
		redemption := strings.Replace(data[2], ",", "", -1)
		outstanding := strings.Replace(data[3], ",", "", -1)
		lastRemain := strings.Replace(data[4], ",", "", -1)
		todayRemain := strings.Replace(data[5], ",", "", -1)
		if data[0] == "融資(交易單位)" {
			marginShort.Margin.TodayNew = todayNew
			marginShort.Margin.Redemption = redemption
			marginShort.Margin.Outstanding = outstanding
			marginShort.Margin.LastRemain = lastRemain
			marginShort.Margin.TodayRemain = todayRemain
		} else if data[0] == "融券(交易單位)" {
			marginShort.Short.TodayNew = todayNew
			marginShort.Short.Redemption = redemption
			marginShort.Short.Outstanding = outstanding
			marginShort.Short.LastRemain = lastRemain
			marginShort.Short.TodayRemain = todayRemain
		} else if data[0] == "融資金額(仟元)" {
			marginShort.MarginValue.TodayNew = todayNew
			marginShort.MarginValue.Redemption = redemption
			marginShort.MarginValue.Outstanding = outstanding
			marginShort.MarginValue.LastRemain = lastRemain
			marginShort.MarginValue.TodayRemain = todayRemain
		} else {
			break
		}
	}

	return marginShort, true
}

func writeIndexQuote(db *sql.DB, date, o, h, l, c, volume, amount, count string) bool {
	sqlString := "DELETE FROM index_values WHERE trade_date='" + date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	_, err := db.Exec(sqlString)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	sqlString = fmt.Sprintf("INSERT INTO index_values VALUES ('%s', 'TAIEX', '%s', '%s', '%s', '%s', '%s', '%s', '%s');",
		date, o, h, l, c, volume, amount, count)
	//fmt.Println(sqlString)
	_, err = db.Exec(sqlString)
	if err != nil {
		log.Println("db.Exec", err)
		return false
	}
	return true
}

func writeIndexInvestor(db *sql.DB, date string, investor IndexInvestor) bool {
	sqlString := "DELETE FROM index_investors WHERE trade_date='" + date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	_, err := db.Exec(sqlString)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	sqlString = fmt.Sprintf("INSERT INTO index_investors VALUES ('%s', 'TAIEX', ", date)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerSelf.Buy)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerSelf.Sell)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerSelf.Difference)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerHedge.Buy)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerHedge.Sell)
	sqlString += fmt.Sprintf("'%s', ", investor.DealerHedge.Difference)
	sqlString += fmt.Sprintf("'%s', ", investor.Trust.Buy)
	sqlString += fmt.Sprintf("'%s', ", investor.Trust.Sell)
	sqlString += fmt.Sprintf("'%s', ", investor.Trust.Difference)
	sqlString += fmt.Sprintf("'%s', ", investor.Foreign.Buy)
	sqlString += fmt.Sprintf("'%s', ", investor.Foreign.Sell)
	sqlString += fmt.Sprintf("'%s', ", investor.Foreign.Difference)
	sqlString += fmt.Sprintf("'%s', ", investor.Total.Buy)
	sqlString += fmt.Sprintf("'%s', ", investor.Total.Sell)
	sqlString += fmt.Sprintf("'%s');", investor.Total.Difference)
	//fmt.Println(sqlString)
	_, err = db.Exec(sqlString)
	if err != nil {
		log.Println("db.Exec", err)
		return false
	}
	return true
}

func writeIndexMarginShort(db *sql.DB, date string, data IndexMarginShort) bool {
	sqlString := "DELETE FROM index_margin_short WHERE trade_date='" + date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	_, err := db.Exec(sqlString)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	sqlString = fmt.Sprintf("INSERT INTO index_margin_short VALUES ('%s', 'TAIEX', ", date)
	sqlString += fmt.Sprintf("'%s', ", data.Margin.TodayNew)
	sqlString += fmt.Sprintf("'%s', ", data.Margin.Redemption)
	sqlString += fmt.Sprintf("'%s', ", data.Margin.Outstanding)
	sqlString += fmt.Sprintf("'%s', ", data.Margin.LastRemain)
	sqlString += fmt.Sprintf("'%s', ", data.Margin.TodayRemain)
	sqlString += fmt.Sprintf("'%s', ", data.Short.TodayNew)
	sqlString += fmt.Sprintf("'%s', ", data.Short.Redemption)
	sqlString += fmt.Sprintf("'%s', ", data.Short.Outstanding)
	sqlString += fmt.Sprintf("'%s', ", data.Short.LastRemain)
	sqlString += fmt.Sprintf("'%s', ", data.Short.TodayRemain)
	sqlString += fmt.Sprintf("'%s', ", data.MarginValue.TodayNew)
	sqlString += fmt.Sprintf("'%s', ", data.MarginValue.Redemption)
	sqlString += fmt.Sprintf("'%s', ", data.MarginValue.Outstanding)
	sqlString += fmt.Sprintf("'%s', ", data.MarginValue.LastRemain)
	sqlString += fmt.Sprintf("'%s');", data.MarginValue.TodayRemain)
	//fmt.Println(sqlString)
	_, err = db.Exec(sqlString)
	if err != nil {
		log.Println("db.Exec", err)
		return false
	}
	return true
}
