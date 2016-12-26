package main

import (
	"fmt"
	"log"
	"strings"
	"net/http"
	"bufio"
	"encoding/csv"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var enc = traditionalchinese.Big5

const urlTSE = "http://www.tse.com.tw/ch/trading/exchange/MI_INDEX/MI_INDEX.php?download=csv&qdate=105/12/02&selectType=01"
//const urlTSE = "http://www.tse.com.tw/en/trading/exchange/MI_INDEX/MI_INDEX.php?download=csv&qdate=2016/11/04&selectType=01"

func main() {
	resp, err := http.Get(urlTSE)
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
	firstBlankLine := true
	lineCount := 0
	out := ""
	for input.Scan() {
		in := strings.TrimSpace(input.Text())
		if len(in) == 0 {
			if firstBlankLine {
				firstBlankLine = false
				continue
			} else {
				break
			}
		}
		if firstBlankLine == false {
			lineCount++
			if lineCount >= 4 {
				out += in + "\n"
			}
		}
	}
	fmt.Print(out)

	fmt.Println("----")

	if lineCount < 2 {
		fmt.Println("No Content")
		return
	}

	cvsr := csv.NewReader(strings.NewReader(out))

	records, err := cvsr.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, record := range records {
		for _, field := range record {
			field = strings.Replace(field, ",", "", -1)
			fmt.Print(strings.TrimSpace(field))
			fmt.Print("\t")
		}
		fmt.Println()
	}
}
