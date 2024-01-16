package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/chzyer/readline"
	_ "github.com/go-sql-driver/mysql"
	"github.com/liushuochen/gotable"
	"io"
	"regexp"
	"strings"
	"time"
)

var username string
var password string
var host string
var port string
var database string
var executeCmd string

var tables [][]rune

func parseFlag() {
	flag.StringVar(&username, "u", "root", "用户名，默认为root")
	flag.StringVar(&password, "p", "", "密码，默认为空")
	flag.StringVar(&host, "h", "localhost", "主机名，默认为localhost")
	flag.StringVar(&port, "P", "3306", "端口号，默认为3306")
	flag.StringVar(&database, "D", "test", "数据库，默认为test")
	flag.StringVar(&executeCmd, "e", "", "数据库，默认为空")
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

func setTables(db *sql.DB) error {
	rows, err := db.Query("show tables")
	if err != nil {
		fmt.Println(err)
		return err
	}

	columns, list := parseRows(rows)
	for _, row := range list {
		tables = append(tables, []rune(row[columns[0]]))
	}
	return nil
}

func queryAny(db *sql.DB, cmdline string) {
	executeSql := cmdline

	compile := regexp.MustCompile(`(.*)\\[json|table]\s*;?`)
	submatch := compile.FindStringSubmatch(cmdline)
	if len(submatch) > 0 {
		executeSql = submatch[1]
	}

	rows, err := db.Query(executeSql)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	columns, results := parseRows(rows)
	if results == nil {
		return
	}

	cmdline = strings.TrimSpace(cmdline)
	if strings.HasSuffix(cmdline, "\\json") {
		printJson(columns, results)
	} else if strings.HasSuffix(cmdline, "\\table") {
		printTable(columns, results)
	} else {
		printSimple(columns, results)
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
	fmt.Println("[")
	var elems []string
	for _, v := range results {
		marshal, _ := json.Marshal(v)
		elems = append(elems, string(marshal))
	}
	println(strings.Join(elems, ",\n"))
	fmt.Println("]")
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

func printSimple(columns []string, results []map[string]string) {
	for _, m := range results {
		var elems []string
		for _, col := range columns {
			elems = append(elems, m[col])
		}
		println(strings.Join(elems, "\t"))
	}
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

var completer = readline.SegmentFunc(func(segment [][]rune, idx int) (cands [][]rune) {
	cands = append(cands, tables...)
	return cands
})

func initReadline() (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		Prompt:          "\033[32m" + database + " »\033[0m ",
		HistoryFile:     "/tmp/readline.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
}

func main() {
	parseFlag()

	db := initDB()
	if db == nil {
		return
	}
	defer db.Close()
	// 立即执行非交互式
	if executeCmd != "" {
		queryAny(db, executeCmd)
		return
	}

	if setTables(db) != nil {
		return
	}

	l, err := initReadline()
	if err != nil {
		panic(err)
	}
	defer l.Close()
	l.CaptureExitSignal()

	for true {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if "quit" == line || "exit" == line {
			return
		}
		if "" != line {
			queryAny(db, line)
		}
	}
}
