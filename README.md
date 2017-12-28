# TWStock
Taiwan Stock Data Crawler

Fetch stock data from Taiwan Stock Exchange and TPEX (OTC).

Reference Data:
https://www.gitbook.com/book/cfw011566/taiwan-stock-data-download-and-transfer/details

URLs
1. Daily Quote
```
const urlTSEDailyQuote = "http://www.twse.com.tw/exchangeReport/MI_INDEX?response=json&date=%4d%02d%02d&type=ALLBUT0999"
const urlOTCDailyQuote = "http://www.tpex.org.tw/web/stock/aftertrading/otc_quotes_no1430/stk_wn1430_download.php?l=zh-tw&d=%d/%02d/%02d&se=EW&s=0,asc,0"
```

