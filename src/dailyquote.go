package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
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
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var enc = traditionalchinese.Big5

const urlTSEDailyQuote = "http://www.twse.com.tw/exchangeReport/MI_INDEX?response=json&date=%4d%02d%02d&type=ALLBUT0999"
const urlOTCDailyQuote = "http://www.tpex.org.tw/web/stock/aftertrading/otc_quotes_no1430/stk_wn1430_download.php?l=zh-tw&d=%d/%02d/%02d&se=EW&s=0,asc,0"
const kMinSize = 1024
const kMinDate = 20000000

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

const deleteSql = "DELETE FROM daily_quotes WHERE trade_date = $1"

type DailyQuote struct {
	Date    string
	Fields  []string        `json:"fields1"`
	Data    [][]string      `json:"data5"`
	Indices [][]string      `json:"data1"`
	Trades  [][]interface{} `json:"data3"`
}

var dailyQuote DailyQuote

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
		quotes, ok := fetchTSEDailyQuotes(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		csvString, ok2 := fetchOTCDailyQuotes(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if ok && ok2 {
			_, err := db.Exec(deleteSql, quotes.Date)
			//log.Println(result)
			if err == nil || err == sql.ErrNoRows {
				//printTSEDailyQuotes(quotes)
				writeTSEDailyQuotes(db, quotes)
				//printOTCDailyQuotes(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day(), csvString)
				writeOTCDailyQuotes(db, quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day(), csvString)
			} else {
				log.Printf("DELETE error = %v\n", err)
			}
		}
		if *flagLastTradeDay {
			break
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func writeTSEDailyQuotes(db *sql.DB, quotes *DailyQuote) bool {
	// "fields5":["證券代號","證券名稱","成交股數","成交筆數","成交金額","開盤價","最高價","最低價","收盤價","漲跌(+/-)","漲跌價差","最後揭示買價","最後揭示買量","最後揭示賣價","最後揭示賣量","本益比"],
	sqlString := "INSERT INTO daily_quotes (trade_date, security_code, trade_volume, trade_count, trade_amount, open_price, highest_price, lowest_price, close_price, last_bid_price, last_bid_volume, last_ask_price, last_ask_volume) VALUES\n"
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
	_, err := db.Exec(sqlString)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func printTSEDailyQuotes(quotes *DailyQuote) {
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

func fetchTSEDailyQuotes(year int, month int, day int) (*DailyQuote, bool) {
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

func fetchOTCDailyQuotes(year, month, day int) (string, bool) {
	var url string

	url = fmt.Sprintf(urlOTCDailyQuote, year-1911, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)

	r := transform.NewReader(resp.Body, enc.NewDecoder())
	input := bufio.NewScanner(r)

	lineCount := 0
	out := ""
	for input.Scan() {
		in := strings.TrimSpace(input.Text())
		if len(in) <= 20 {
			continue
		}
		lineCount++
		if lineCount > 4 {
			out += in + "\n"
		}
	}

	if lineCount < 2 {
		return "", false
	}

	return out, true
}

func printOTCDailyQuotes(year, month, day int, csvString string) {
	log.Println(year, month, day)

	csvr := csv.NewReader(strings.NewReader(csvString))

	records, err := csvr.ReadAll()
	if err == nil {
		for _, record := range records {
			for _, field := range record {
				field = strings.Replace(field, ",", "", -1)
				fmt.Print(strings.TrimSpace(field))
				fmt.Print("\t")
			}
			fmt.Println()
		}
	}
}

func writeOTCDailyQuotes(db *sql.DB, year, month, day int, csvString string) bool {
	csvr := csv.NewReader(strings.NewReader(csvString))

	records, err := csvr.ReadAll()
	if err != nil {
		log.Println(err)
		return false
	}

	quoteDate := fmt.Sprintf("%04d/%02d/%02d", year, month, day)

	// 代號,名稱,收盤 ,漲跌,開盤 ,最高 ,最低,成交股數  , 成交金額(元), 成交筆數 ,最後買價,最後賣價,發行股數 ,次日漲停價 ,次日跌停價
	// sqlString := "INSERT INTO daily_quotes (trade_date, security_code, trade_volume, trade_count, trade_amount, open_price, highest_price, lowest_price, close_price, last_bid_price, last_bid_volume, last_ask_price, last_ask_volume) VALUES\n"
	sqlString := "INSERT INTO daily_quotes (trade_date, security_code, close_price, open_price, highest_price, lowest_price, trade_volume, trade_amount, trade_count, last_bid_price, last_bid_volume, last_ask_price, last_ask_volume) VALUES\n"

	for _, record := range records {
		sqlString += fmt.Sprintf("('%s',", quoteDate)

		for i, field := range record {
			if i == 1 || i == 3 {
				continue
			}

			if strings.Contains(field, "--") || len(field) == 0 {
				sqlString += " null"
			} else {
				sqlString += " '" + strings.Replace(field, ",", "", -1) + "'"
			}
			sqlString += ","
			if i == 10 {
				sqlString += " 0,"
			}
			if i == 11 {
				sqlString += " 0"
				break
			}
		}
		sqlString += "),\n"
	}

	sqlString = strings.TrimRight(sqlString, ",\n") + ";"
	//fmt.Println(sqlString)
	_, err = db.Exec(sqlString)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
