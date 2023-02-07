package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/liushuochen/gotable"
	"os"
	"regexp"
	"strings"
	"time"
)

var username string
var password string
var host string
var port string
var database string

func parseFlag() {
	flag.StringVar(&username, "u", "root", "用户名，默认为root")
	flag.StringVar(&password, "p", "", "密码，默认为空")
	flag.StringVar(&host, "h", "localhost", "主机名，默认为localhost")
	flag.StringVar(&port, "P", "3306", "端口号，默认为3306")
	flag.StringVar(&database, "D", "test", "数据库，默认为test")
	flag.Parse()
}

func initDB() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, database)
	DB, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Open mysql failed,err:%v\n", err)
		return nil
	}
	DB.SetConnMaxLifetime(100 * time.Second) //最大连接周期，超过时间的连接就close
	DB.SetMaxOpenConns(100)                  //设置最大连接数
	DB.SetMaxIdleConns(16)                   //设置闲置连接数

	if err = DB.Ping(); err != nil {
		fmt.Println(err)
		return nil
	}
	return DB
}

func queryAny(db *sql.DB, cmdline string) {
	sql := cmdline
	jsonFmt := false

	compile := regexp.MustCompile(`(.*)\\json\s*;?`)
	submatch := compile.FindStringSubmatch(cmdline)
	if len(submatch) > 0 {
		jsonFmt = true
		sql = submatch[1]
	}

	if strings.HasSuffix(cmdline, "\\json") {
		jsonFmt = true
	}

	rows, err := db.Query(sql)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	columns, results := parseRows(rows)
	if results == nil {
		return
	}

	if jsonFmt {
		printJson(columns, results)
	} else {
		printTable(columns, results)
	}

}

func parseRows(rows *sql.Rows) ([]string, []map[string]string) {
	columns, _ := rows.Columns()

	values := make([][]byte, len(columns))
	scans := make([]interface{}, len(columns))

	for i := range values {
		scans[i] = &values[i]
	}

	var results []map[string]string

	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			fmt.Println("查询失败，", err)
			return columns, nil
		}

		row := make(map[string]string)
		for k, v := range values {
			key := columns[k]
			row[key] = string(v)
		}

		results = append(results, row)
	}
	return columns, results
}

func printJson(_ []string, results []map[string]string) {
	for _, v := range results {
		marshal, _ := json.Marshal(v)
		fmt.Println(string(marshal))
	}
}

func printTable(columns []string, results []map[string]string) {
	table, err := gotable.Create(columns...)
	if err != nil {
		fmt.Println(err)
		return
	}
	table.AddRows(results)
	fmt.Println(table)
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return ""
	}
	return string(line)
}

func main() {
	parseFlag()

	db := initDB()
	if db == nil {
		return
	}
	defer db.Close()

	for true {
		fmt.Printf("%s> ", database)
		cmdline := readLine()
		if "quit" == cmdline || "exit" == cmdline {
			return
		}
		if "" != cmdline {
			queryAny(db, cmdline)
		}
	}
}
