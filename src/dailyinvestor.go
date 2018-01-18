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

const urlTSEDailyInvestor = "http://www.twse.com.tw/fund/T86?response=json&date=%4d%02d%02d&selectType=ALLBUT0999"
const urlOTCDailyInvestor = "http://www.tpex.org.tw/web/stock/3insti/daily_trade/3itrade_hedge_download.php?l=zh-tw&se=EW&t=D&d=%d/%02d/%02d&s=0,asc"
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
		quoteDate = time.Date(toDate/10000, time.Month(toDate%10000/100), toDate%100, 1, 0, 0, 0, local)
	}
	beginDate = time.Date(fromDate/10000, time.Month(fromDate%10000/100), fromDate%100, 0, 0, 0, 0, local)
	log.Println("beginDate = ", beginDate)
	for quoteDate.After(beginDate) {
		log.Println(quoteDate)
		time.Sleep(1 * time.Second)
		// quotes, ok := fetchtDailyInvestors(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		csvString, ok := fetchOTCDailyInvestors(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day())
		if ok {
			// writeDailyInvestors(db, quotes)
			// printDailyInvestors(quotes)
			printOTCDailyInvestors(quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day(), csvString)
			writeOTCDailyInvestors(db, quoteDate.Year(), int(quoteDate.Month()), quoteDate.Day(), csvString)
			if *flagLastTradeDay {
				break
			}
		}
		quoteDate = quoteDate.AddDate(0, 0, -1)
	}
}

func writeTSEDailyInvestors(db *sql.DB, quotes *DailyInvestor) bool {
	sqlString := "DELETE FROM daily_investors WHERE trade_date='" + quotes.Date + "';"
	result, err := db.Exec(sqlString)
	log.Println(result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DELETE error = %v\n", err)
		return false
	}

	// "fields":["證券代號","證券名稱","外陸資買進股數(不含外資自營商)","外陸資賣出股數(不>含外資自營商)","外陸資買賣超股數(不含外資自營商)","外資自營商買進股數","外資自營商賣出股數","外資自營商買賣超股數","投信買進股數","投信賣出股數","投信買賣超股數","自營商買賣超股數","自營商買進股數(自行買賣)","自營商賣出股數(自行買賣)","自營商買賣超股數(自行買賣)","自營商買進股數(避險)","自營商賣出股數(避險)","自營>商買賣超股數(避險)","三大法人買賣超股數"]
	sqlString = "INSERT INTO daily_investors (trade_date, security_code, foreign_buy, foreign_sell, foreign_diff, foreign_self_buy, foreign_self_sell, foreign_self_diff, trust_buy, trust_sell, trust_diff, dealer_diff, dealer_self_buy, dealer_self_sell, dealer_self_diff, dealer_hedge_buy, dealer_hedge_sell, dealer_hedge_diff, investors_diff) VALUES\n"
	for _, quote := range quotes.Data {
		sqlString += fmt.Sprintf("('%s',", quotes.Date)
		quoteDateInt, err := strconv.Atoi(quotes.Date)
		if err != nil {
			log.Println("writeDailyInvestors: ", err)
		}
		for i := 0; i < len(quote); i++ {
			if i == 1 {
				continue
			}
			if quoteDateInt >= 20171218 {
				if strings.Contains(quote[i], "--") || len(quote[i]) == 0 {
					sqlString += " null"
				} else {
					sqlString += " '" + strings.Replace(quote[i], ",", "", -1) + "'"
				}
				if i != 18 {
					sqlString += ","
				}
			} else {
				if i == 5 {
					sqlString += " '0', '0', '0',"
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
		}
		sqlString += "),\n"
	}
	sqlString = strings.TrimRight(sqlString, ",\n") + ";"
	//fmt.Println(sqlString)
	result, err = db.Exec(sqlString)
	if err != nil {
		log.Println("writeDailyInvestors: ", err)
		return false
	}
	return true
}

func printTSEDailyInvestors(quotes *DailyInvestor) {
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

func fetchTSEDailyInvestors(year int, month int, day int) (*DailyInvestor, bool) {
	var url string
	var contents []byte

	url = fmt.Sprintf(urlTSEDailyInvestor, year, month, day)
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

	err = json.Unmarshal(contents, &dailyInvestor)
	if err != nil {
		log.Println("json unmarshal: ", err)
		return nil, false
	}

	return &dailyInvestor, true
}

// OTC

func fetchOTCDailyInvestors(year, month, day int) (string, bool) {
	var url string

	url = fmt.Sprintf(urlOTCDailyInvestor, year-1911, month, day)
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
		if lineCount > 2 {
			out += in + "\n"
		}
	}

	if lineCount < 2 {
		return "", false
	}

	return out, true
}

func printOTCDailyInvestors(year, month, day int, csvString string) {
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

func writeOTCDailyInvestors(db *sql.DB, year, month, day int, csvString string) bool {
	csvr := csv.NewReader(strings.NewReader(csvString))

	records, err := csvr.ReadAll()
	if err != nil {
		log.Println(err)
		return false
	}

	quoteDate := fmt.Sprintf("%04d/%02d/%02d", year, month, day)
	quoteDateInt := year*10000 + month*100 + day

	sqlString := "INSERT INTO daily_investors (trade_date, security_code, foreign_buy, foreign_sell, foreign_diff, foreign_self_buy, foreign_self_sell, foreign_self_diff, trust_buy, trust_sell, trust_diff, dealer_self_buy, dealer_self_sell, dealer_self_diff, dealer_hedge_buy, dealer_hedge_sell, dealer_hedge_diff, dealer_diff, investors_diff) VALUES\n"

	for _, record := range records {
		sqlString += fmt.Sprintf("('%s',", quoteDate)

		if quoteDateInt >= 20180115 {
			// After 2018-01-15
			// 代號,名稱,外資及陸資(不含外資自營商)-買進股數,外資及陸資(不含外資自營商)-賣出股數,外資及陸資(不含外資自營商)-買賣超股數,外資自營商-買進股數,外資自營商-賣出股數,外資自營商-買賣超股數,外資及陸資-買進股數,外資及陸資-賣出股數,外資及陸資-買賣超股數,投信-買進股數,投信-賣出股數,投信-買賣超股數,自營商(自行買賣)-買進股數,自營商(自行買賣)-賣出股數,自營商(自行買賣)-買賣超股數,自營商(避險)-買進股數,自營商(避險)-賣出股數,自營商(避險)-買賣超股數,自營商-買進股數,自營商-賣出股數,自營商-買賣超股數,三大法人買賣超股數合計
			for i, field := range record {
				if i == 1 || i == 8 || i == 9 || i == 10 || i == 20 || i == 21 {
					continue
				}
				if strings.Contains(field, "--") || len(field) == 0 {
					sqlString += " null"
				} else {
					sqlString += " '" + strings.Replace(field, ",", "", -1) + "'"
				}
				if i != 23 {
					sqlString += ","
				}
			}
			sqlString += "),\n"
		} else {
			// 代號,名稱,外資及陸資買股數,外資及陸資賣股數,外資及陸資淨買股數,投信買進股數,投信賣股數,投信淨買股數,自營淨買股數,自營商(自行買賣)買股數,自營商(自行買賣)賣股數,自營商(自行買賣)淨買股數,自營商(避險)買股數,自營商(避險)賣股數,自營商(避險)淨買股數,三大法人買賣超股數
			var dealer_diff string
			for i, field := range record {
				if i == 1 {
					continue
				}
				if i == 8 {
					if strings.Contains(field, "--") || len(field) == 0 {
						dealer_diff = "null"
					} else {
						dealer_diff = strings.Replace(field, ",", "", -1)
					}
					continue
				}
				if strings.Contains(field, "--") || len(field) == 0 {
					sqlString += " null"
				} else {
					sqlString += " '" + strings.Replace(field, ",", "", -1) + "'"
				}
				if i == 4 {
					sqlString += ", '0', '0', '0'"
				}
				if i == 14 {
					sqlString += ", '" + dealer_diff + "'"
				}
				if i != 15 {
					sqlString += ","
				}
			}
			sqlString += "),\n"
		}
	}

	sqlString = strings.TrimRight(sqlString, ",\n") + ";"
	fmt.Println(sqlString)
	_, err = db.Exec(sqlString)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
