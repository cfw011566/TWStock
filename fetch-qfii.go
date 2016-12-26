package main

import (
	"fmt"
	"log"
	"strings"
	"net/http"
	"net/url"
	"bufio"
	"encoding/csv"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var enc = traditionalchinese.Big5

const urlTSE = "http://www.tse.com.tw/ch/trading/fund/T86/T86.php"

func main() {
	v := url.Values{}
	v.Set("download", "csv")
	v.Set("qdate", "105/11/06")
	v.Set("select2", "01")
	v.Set("sorting", "by_stkno")
	resp, err := http.PostForm(urlTSE, v)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("----")
	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Status)

	fmt.Println("----")
	r := transform.NewReader(resp.Body, enc.NewDecoder())
	input := bufio.NewScanner(r)
//	input := bufio.NewScanner(resp.Body)
	out := ""
	lineCount := 0
	for input.Scan() {
		in := strings.TrimSpace(input.Text())
		if strings.Contains(in, "說明") {
			break
		}
		lineCount++
		if lineCount > 2 {
			in = strings.TrimRight(in, ",")
			out += strings.TrimLeft(in, "=") + "\n"
		}
	}
	fmt.Print(out)

	fmt.Println("----")

	if strings.Contains(out, "查無資料") {
		return
	}

	cvsr := csv.NewReader(strings.NewReader(out))

	records, err := cvsr.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, record := range records {
		for _, field := range record {
			fmt.Print(strings.TrimSpace(field))
			fmt.Print(",")
		}
		fmt.Println()
	}
}
