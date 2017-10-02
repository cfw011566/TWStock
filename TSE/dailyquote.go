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

const urlTSEDailyQuote = "http://www.twse.com.tw/exchangeReport/MI_INDEX?response=json&date=%4d%02d%02d&type=ALLBUT0999"
const urlTSEDailySubTrade = "http://www.twse.com.tw//exchangeReport/BFIAMU?response=json&date=%4d%02d%02d"
const kMinSize = 1024
const kMinDate = 20000000

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

type DailyQuote struct {
	Date    string
	Fields  []string        `json:"fields1"`
	Data    [][]string      `json:"data5"`
	Indices [][]string      `json:"data1"`
	Trades  [][]interface{} `json:"data3"`
}

type DailySubTrade struct {
	Date   string
	Trades [][]string `json:"data"`
}

var dailyQuote DailyQuote
var dailySubTrade DailySubTrade
var indexCodes map[string]string

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

	if indexCodes = readIndexCode(db); len(indexCodes) == 0 {
		return
	}

	today := time.Now()
	fromDate := *flagFromDate
	toDate := *flagToDate
	if fromDate == 0 || fromDate < kMinDate {
		fromDate, err = fetchLastTradeDate(db)
		if err != nil {
			fromDate = today.Year()*10000 + int(today.Month())*100 + today.Day()
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
		quotes, ok := fetchDailyQuotes(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		subTrades, ok2 := fetchDailySubTrades(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if ok && ok2 {
			//printDailyQuotes(quotes)
			//printDailySubTrades(subTrades)
			writeDailyQuotes(db, quotes)
			writeDailyIndices(db, quotes, subTrades)
			if *flagLastTradeDay {
				break
			}
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func writeDailyQuotes(db *sql.DB, quotes *DailyQuote) bool {
	//	var lastInsertId string

	sqlString := "DELETE FROM daily_quotes WHERE trade_date='" + quotes.Date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	result, err := db.Exec(sqlString)
	log.Println(result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	// "fields5":["證券代號","證券名稱","成交股數","成交筆數","成交金額","開盤價","最高價","最低價","收盤價","漲跌(+/-)","漲跌價差","最後揭示買價","最後揭示買量","最後揭示賣價","最後揭示賣量","本益比"],
	sqlString = "INSERT INTO daily_quotes (trade_date, security_code, trade_volume, trade_count, trade_amount, open_price, highest_price, lowest_price, close_price, last_bid_price, last_bid_volume, last_ask_price, last_ask_volume) VALUES\n"
	for _, quote := range quotes.Data {
		sqlString += fmt.Sprintf("('%s',", quotes.Date)
		for i := 0; i < len(quote); i++ {
			if i == 1 || i == 9 || i == 10 || i == 15 {
				continue
			}
			if strings.Contains(quote[i], "--") || len(quote[i]) == 0 {
				sqlString += " null"
			} else {
				sqlString += " '" + strings.Replace(quote[i], ",", "", -1) + "'"
			}
			if i != 14 {
				sqlString += ","
			}
		}
		sqlString += "),\n"
	}
	sqlString = strings.TrimRight(sqlString, ",\n") + ";"
	//fmt.Println(sqlString)
	result, err = db.Exec(sqlString)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func printDailyQuotes(quotes *DailyQuote) {
	log.Println(quotes.Date)
	for _, quote := range quotes.Data {
		for _, field := range quote {
			fmt.Print(strings.Replace(field, ",", "", -1))
			fmt.Print("\t")
		}
		fmt.Println()
	}
}

func fetchLastTradeDate(db *sql.DB) (int, error) {
	var row1 string
	sqlString := "SELECT to_char(MAX(trade_date)+interval '1 day', 'YYYYMMDD') FROM daily_quotes"
	err := db.QueryRow(sqlString).Scan(&row1)
	if err != nil {
		return 0, err
	}
	log.Println("the day after latest trade day = ", row1)

	return strconv.Atoi(row1)
}

func fetchDailyQuotes(year int, month int, day int) (*DailyQuote, bool) {
	var url string
	var contents []byte

	url = fmt.Sprintf(urlTSEDailyQuote, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	if len(contents) < kMinSize {
		return nil, false
	}

	err = json.Unmarshal(contents, &dailyQuote)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return nil, false
	}

	return &dailyQuote, true
}

func fetchDailySubTrades(year int, month int, day int) (*DailySubTrade, bool) {
	var url string
	var contents []byte

	url = fmt.Sprintf(urlTSEDailySubTrade, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	if len(contents) < kMinSize {
		return nil, false
	}

	err = json.Unmarshal(contents, &dailySubTrade)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return nil, false
	}

	return &dailySubTrade, true
}

func writeDailyIndices(db *sql.DB, quotes *DailyQuote, subTrades *DailySubTrade) bool {
	//	var lastInsertId string

	sqlString := "DELETE FROM daily_indices WHERE trade_date='" + quotes.Date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	result, err := db.Exec(sqlString)
	log.Println(result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	sqlString = "INSERT INTO daily_indices (trade_date, security_code, index_value, trade_volume, trade_amount, trade_count) VALUES\n"
	for name, code := range indexCodes {
		//sqlString += fmt.Sprintf("('%s','%s',", quotes.Date, code)
		if code == "index01" {
			indexValue, ok := getIndexValue(name, quotes)
			tradeVolume, tradeAmount, tradeCount, ok2 := getIndexTrade(quotes)
			if !ok || !ok2 {
				continue
			}
			sqlString += fmt.Sprintf("('%s','%s',", quotes.Date, code)
			sqlString += fmt.Sprintf("'%s','%s','%s','%s'),\n", indexValue, tradeVolume, tradeAmount, tradeCount)
		} else if strings.HasPrefix(code, "index") {
			indexValue, ok := getIndexValue(name, quotes)
			if !ok {
				continue
			}
			sqlString += fmt.Sprintf("('%s','%s',", quotes.Date, code)
			sqlString += fmt.Sprintf("'%s',null,null,null),\n", indexValue)
		} else {
			indexValue, ok := getIndexValue(name, quotes)
			tradeVolume, tradeAmount, tradeCount, ok2 := getSubTrade(name, subTrades)
			if !ok || !ok2 {
				continue
			}
			sqlString += fmt.Sprintf("('%s','%s',", quotes.Date, code)
			sqlString += fmt.Sprintf("'%s','%s','%s','%s'),\n", indexValue, tradeVolume, tradeAmount, tradeCount)
		}
	}
	sqlString = strings.TrimRight(sqlString, ",\n") + ";"
	//fmt.Println(sqlString)
	result, err = db.Exec(sqlString)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func printDailySubTrades(quotes *DailySubTrade) {
	log.Println(quotes.Date)
	for _, quote := range quotes.Trades {
		for _, field := range quote {
			fmt.Print(strings.Replace(field, ",", "", -1))
			fmt.Print("\t")
		}
		fmt.Println()
	}
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

func getIndexValue(name string, quotes *DailyQuote) (string, bool) {
	if strings.HasPrefix(name, "觀光") {
		name = strings.Replace(name, "事業", "", -1)
	}
	for _, index := range quotes.Indices {
		if index[0] == name {
			return strings.Replace(index[1], ",", "", -1), true
		}
	}
	return "", false
}

func getIndexTrade(quotes *DailyQuote) (string, string, string, bool) {
	for _, trade := range quotes.Trades {
		if strings.HasPrefix(trade[0].(string), "總計") {
			tradeAmount := strings.Replace(trade[1].(string), ",", "", -1)
			tradeVolume := strings.Replace(trade[2].(string), ",", "", -1)
			tradeCount := strings.Replace(trade[3].(string), ",", "", -1)
			return tradeVolume, tradeAmount, tradeCount, true
		}
	}
	return "", "", "", false
}

func getSubTrade(name string, quotes *DailySubTrade) (string, string, string, bool) {
	for _, trade := range quotes.Trades {
		if strings.HasPrefix(trade[0], name) {
			tradeAmount := strings.Replace(trade[1], ",", "", -1)
			tradeVolume := strings.Replace(trade[2], ",", "", -1)
			tradeCount := strings.Replace(trade[3], ",", "", -1)
			return tradeVolume, tradeAmount, tradeCount, true
		}
	}
	return "", "", "", false
}
