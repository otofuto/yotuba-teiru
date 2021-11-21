package database

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	//"github.com/joho/godotenv"
)

func Connect() *sql.DB {
	//godotenv.Load()

	connectionstring := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASS") + "@" + os.Getenv("DB_HOST") + "/" + os.Getenv("DB_NAME") + ""
	db, err := sql.Open("mysql", connectionstring)
	if err != nil {
		fmt.Println(connectionstring)
		panic(err.Error)
	}

	return db
}

func Escape(str string) string {
	ret := strings.Replace(str, "\\", "\\\\", -1)
	ret = strings.Replace(ret, "\"", "\\\"", -1)
	ret = strings.Replace(ret, "'", "\\'", -1)
	ret = strings.Replace(ret, "\t", "\\t", -1)
	ret = strings.Replace(ret, "\r", "\\r", -1)
	ret = strings.Replace(ret, "\n", "\\n", -1)

	return ret
}

func Int64ToInt(i int64) int {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0
	} else {
		return int(i)
	}
}
