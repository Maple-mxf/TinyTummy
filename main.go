package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Record struct {
	ID         string
	StartTime  time.Time
	EndTime    time.Time
	MilkPowder string
	Water      int
	WaterAfter int
}

type Report struct {
	TotalMilkPowder int
	TotalWater      int
	MaxInterval     int
	MinInterval     int
}

var (
	tmpl      *template.Template
	db        *sql.DB
	powderMap = map[string]float64{"半勺": 0.5, "一勺": 1, "两勺": 2, "三勺": 3, "四勺": 4, "五勺": 5}
)

func main() {
	var err error
	hostAndPort := os.Getenv("MYSQL_ADDRESS")
	pwd := os.Getenv("MYSQL_PASSWORD")
	user := os.Getenv("MYSQL_USERNAME")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/feeding?parseTime=true", user, pwd, hostAndPort)
	if dsn == "" {
		log.Fatal("Environment variable MYSQL_DSN is not set.")
	}
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	tmpl = template.Must(template.New("index.html").Funcs(template.FuncMap{
		"now": func() string {
			return time.Now().Format("2006-01-02T15:04")
		},
	}).ParseFiles("templates/index.html"))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)
	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	recent := getRecentRecords()
	all := getAllRecords()
	stats := generateReport(all)
	tmpl.Execute(w, map[string]interface{}{
		"Recent": recent,
		"All":    all,
		"Stats":  stats,
	})
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, _ := time.Parse("2006-01-02T15:04", r.FormValue("start_time"))
	end, _ := time.Parse("2006-01-02T15:04", r.FormValue("end_time"))
	milk := r.FormValue("milk_powder")
	water := atoi(r.FormValue("water"))
	waterAfter := atoi(r.FormValue("water_after"))
	id := start.Format("2006-01-02T15:04")

	_, err := db.Exec("INSERT INTO records (id, start_time, end_time, milk_powder, water, water_after) VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE end_time=?, milk_powder=?, water=?, water_after=?",
		id, start, end, milk, water, waterAfter, end, milk, water, waterAfter)
	if err != nil {
		log.Println("Insert error:", err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getRecentRecords() []Record {
	since := time.Now().AddDate(0, 0, -2)
	rows, err := db.Query("SELECT id, start_time, end_time, milk_powder, water, water_after FROM records WHERE start_time >= ? ORDER BY start_time DESC", since)
	if err != nil {
		log.Println("Query recent error:", err)
		return nil
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		rec := Record{}
		rows.Scan(&rec.ID, &rec.StartTime, &rec.EndTime, &rec.MilkPowder, &rec.Water, &rec.WaterAfter)
		records = append(records, rec)
	}
	return records
}

func getAllRecords() []Record {
	rows, err := db.Query("SELECT id, start_time, end_time, milk_powder, water, water_after FROM records ORDER BY start_time DESC")
	if err != nil {
		log.Println("Query all error:", err)
		return nil
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		rec := Record{}
		rows.Scan(&rec.ID, &rec.StartTime, &rec.EndTime, &rec.MilkPowder, &rec.Water, &rec.WaterAfter)
		records = append(records, rec)
	}
	return records
}

func generateReport(records []Record) map[string]Report {
	result := make(map[string]Report)
	daily := make(map[string][]Record)
	for _, r := range records {
		date := r.StartTime.Format("2006-01-02")
		daily[date] = append(daily[date], r)
	}

	for date, recs := range daily {
		var sumMilk float64
		sumWater := 0
		var intervals []int

		sort.Slice(recs, func(i, j int) bool {
			return recs[i].StartTime.Before(recs[j].StartTime)
		})

		for i := 0; i < len(recs); i++ {
			sumMilk += powderMap[recs[i].MilkPowder]
			sumWater += recs[i].Water + recs[i].WaterAfter
			if i > 0 {
				delta := recs[i].StartTime.Sub(recs[i-1].EndTime).Minutes()
				intervals = append(intervals, int(delta))
			}
		}

		maxInt, minInt := 0, 0
		if len(intervals) > 0 {
			sort.Ints(intervals)
			minInt, maxInt = intervals[0], intervals[len(intervals)-1]
		}

		result[date] = Report{
			TotalMilkPowder: int(sumMilk),
			TotalWater:      sumWater,
			MaxInterval:     maxInt,
			MinInterval:     minInt,
		}
	}
	return result
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
