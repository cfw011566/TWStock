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

const urlTSEDailyInvestor = "http://www.twse.com.tw/fund/T86?response=json&date=%4d%02d%02d&selectType=ALLBUT0999"
const kMinSize = 1024
const kMinDate = 20000000

const (
	DB_USER     = "stock"
	DB_PASSWORD = "test"
	DB_NAME     = "stock"
	DB_HOST     = "data.example.com"
)

type DailyInvestor struct {
	Date  string
	Field []string   `json:"fields"`
	Data  [][]string `json:"data"`
}

var dailyInvestor DailyInvestor

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
		quoteDate = time.Date(toDate/10000, time.Month(toDate%1000/100), toDate%100, 1, 0, 0, 0, local)
	}
	beginDate = time.Date(fromDate/10000, time.Month(fromDate%1000/100), fromDate%100, 0, 0, 0, 0, local)
	log.Println("beginDate = ", beginDate)
	for quoteDate.After(beginDate) {
		log.Println(quoteDate)
		ok, quotes := getDailyInvestors(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if ok {
			writeDailyInvestors(db, quotes)
			//printDailyInvestors(quotes)
			if *flagLastTradeDay {
				break
			}
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func writeDailyInvestors(db *sql.DB, quotes *DailyInvestor) bool {
	sqlString := "DELETE FROM daily_investors WHERE trade_date='" + quotes.Date + "';"
	result, err := db.Exec(sqlString)
	log.Println(result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	// "fields":["證券代號","證券名稱","外資買進股數","外資賣出股數","外資買賣超股數","投信買進股數","投信賣出股數","投信買賣超股數","自營商買賣超股數","自營商買進股數(自行買賣)","自營商賣出股數(自行買賣)","自營商買賣超股數(自行買賣)","自營商買進股數(避險)","自營商賣出股數(避險)","自營商買賣超股數(避險)","三大法人買賣超股數"
	sqlString = "INSERT INTO daily_investors (trade_date, security_code, foreign_buy, foreign_sell, foreign_diff, trust_buy, trust_sell, trust_diff, dealer_diff, dealer_self_buy, dealer_self_sell, dealer_self_diff, dealer_hedge_buy, dealer_hedge_sell, dealer_hedge_diff, investors_diff) VALUES\n"
	for _, quote := range quotes.Data {
		sqlString += fmt.Sprintf("('%s',", quotes.Date)
		for i := 0; i < len(quote); i++ {
			if i == 1 {
				continue
			}
			if strings.Contains(quote[i], "--") || len(quote[i]) == 0 {
				sqlString += " null"
			} else {
				sqlString += " '" + strings.Replace(quote[i], ",", "", -1) + "'"
			}
			if i != 15 {
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

func printDailyInvestors(quotes *DailyInvestor) {
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
	sqlString := "SELECT to_char(MAX(trade_date)+interval '1 day', 'YYYYMMDD') FROM daily_investors"
	err := db.QueryRow(sqlString).Scan(&row1)
	if err != nil {
		return 0, err
	}
	log.Println("the day after latest trade day = ", row1)

	return strconv.Atoi(row1)
}

func getDailyInvestors(year int, month int, day int) (bool, *DailyInvestor) {
	var url string
	var contents []byte

	url = fmt.Sprintf(urlTSEDailyInvestor, year, month, day)
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false, nil
	}

	log.Println("----")
	log.Println(resp.StatusCode)
	log.Println(resp.Status)
	log.Println("Body len = ", len(contents))

	if len(contents) < kMinSize {
		return false, nil
	}

	err = json.Unmarshal(contents, &dailyInvestor)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return false, nil
	}

	return true, &dailyInvestor
}
