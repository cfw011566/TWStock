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

const urlTSEDailyMarginShort = "http://www.twse.com.tw/exchangeReport/MI_MARGN?response=json&date=%4d%02d%02d&selectType=ALL"
const kMinSize = 1024
const kMinDate = 20000000

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

type DailyMarginShort struct {
	Date   string
	Fields []string   `json:"fields"`
	Data   [][]string `json:"data"`
}

var dailyMarginShort DailyMarginShort

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
		fromDate, err = getLastTradeDate(db)
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
		quotes, ok := getDailyMarginShort(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if ok {
			//printDailyMarginShort(quotes)
			writeDailyMarginShort(db, quotes)
			if *flagLastTradeDay {
				break
			}
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func writeDailyMarginShort(db *sql.DB, quotes *DailyMarginShort) bool {
	//	var lastInsertId string

	sqlString := "DELETE FROM daily_margin_short WHERE trade_date='" + quotes.Date + "';"
	//err := db.QueryRow(sqlString).Scan(&lastInsertId)
	result, err := db.Exec(sqlString)
	log.Println(result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	// "fields":["股票代號","股票名稱","買進","賣出","現金償還","前日餘額","今日餘額","限額","買進","賣出","現金償還","前日餘額","今日餘額","限額","資券互抵","註記"]
	sqlString = "INSERT INTO daily_margin_short (trade_date, security_code, margin_new, margin_redemption, margin_outstanding, margin_last_remain, margin_remain, margin_limit, short_redemption, short_new, short_outstanding, short_last_remain, short_remain, short_limit, margin_and_short) VALUES\n"
	for _, quote := range quotes.Data {
		sqlString += fmt.Sprintf("('%s',", quotes.Date)
		for i := 0; i < len(quote); i++ {
			if i == 1 || i == 15 {
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

func printDailyMarginShort(quotes *DailyMarginShort) {
	log.Println(quotes.Date)
	for _, quote := range quotes.Data {
		for _, field := range quote {
			fmt.Print(strings.Replace(field, ",", "", -1))
			fmt.Print("\t")
		}
		fmt.Println()
	}
}

func getLastTradeDate(db *sql.DB) (int, error) {
	var row1 string
	sqlString := "SELECT to_char(MAX(trade_date)+interval '1 day', 'YYYYMMDD') FROM daily_margin_short"
	err := db.QueryRow(sqlString).Scan(&row1)
	if err != nil {
		return 0, err
	}
	log.Println("the day after latest trade day = ", row1)

	return strconv.Atoi(row1)
}

func getDailyMarginShort(year int, month int, day int) (*DailyMarginShort, ok) {
	var url string
	var contents []byte

	url = fmt.Sprintf(urlTSEDailyMarginShort, year, month, day)
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

	err = json.Unmarshal(contents, &dailyMarginShort)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return nil, false
	}

	return &dailyMarginShort, true
}
