package main

import (
	"context"
	"strings"
	"fmt"
	"os"
	"time"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
)

func main() {

	staticDir, isSet := os.LookupEnv("STATIC_DIR")

	if !isSet {
		staticDir = "./static"
	}

	fs := http.FileServer(http.Dir(staticDir))

	router := httprouter.New()
	router.GET("/api/readings", readings)

	http.Handle("/api/", router)
	http.Handle("/", fs)

	log.Println("Starting service on port :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

var possibleMetrics = map[string]bool {
	"power-consumption": true,
	"power-consumption-phase-1": true,
	"power-consumption-phase-2": true,
	"power-consumption-phase-3": true,
	"power-consumed-tariff-1": true,
	"power-consumed-tariff-2": true,
	"power-delivery": true,
	"power-delivery-phase-1": true,
	"power-delivery-phase-2": true,
	"power-delivery-phase-3": true,
	"power-delivered-tariff-1": true,
	"power-delivered-tariff-2": true,
}

func determineTableName(duration int64) string {
	s := duration / int64(time.Second)
	if s > 2 * 366 * 24 * 60 * 60 {
		return "aggregate_value_1w"
	} else if s > 8 * 30 * 24 * 60 * 60 {
		return "aggregate_value_1d"
	} else if s > 29 * 25 * 60 * 60 {
		return "aggregate_value_3h"
	} else if s > 6 * 25 * 60 * 60 {
		return "aggregate_value_30m"
	} else {
		return "aggregate_value_300"
	}
}

func readings(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	flusher, _ := w.(http.Flusher)
	w.Header().Add("X-Content-Type-Options", "nosniff")

	metrics := r.URL.Query()["metric"]

	// Controleer lengte van metrics
	if len(metrics) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing metrics"))
		return
	}

	var unknownMetrics []string
	for i := 0; i < len(metrics); i++ {
		if _, present := possibleMetrics[metrics[i]]; !present {
			unknownMetrics = append(unknownMetrics, metrics[i])
		}
	}

	if len(unknownMetrics) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unknown metrics " + strings.Join(unknownMetrics, ", ")))
		return
	}

	durationString := r.URL.Query().Get("duration")

	var fromTimestamp time.Time
	var duration *Duration
	if len(durationString) > 0 {
		var err error
		duration, err = FromString(durationString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Wrongly formatted duration: " + durationString))
			return
		}
	} else {
		duration = &Duration{ Days: 1 }
	}

	toTimestamp := time.Now()
	fromTimestamp = toTimestamp.Add(-duration.ToDuration())

	tableName := determineTableName(int64(duration.ToDuration()))

	// postgres://YourUserName:YourPassword@YourHost:5432/YourDatabase
	config, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Errorf("Unable to parse connection string: %v", err)
	}
	config.Logger = logrusadapter.NewLogger(log.New())

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Errorf("Unable to establish connection: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	metricsString := "'" + strings.Join(metrics, "','") + "'"

	sql := "select metric_id, m.name, timestamp, count, sum, sum_squares, min, max " +
		"from " + tableName + " av " +
		"join metric m on" +
		"  m.id = av.metric_id " +
		"where" +
		"  m.name in (" + metricsString + ") " +
		"  and timestamp BETWEEN $1 AND $2 " +
		"order by av.metric_id, av.timestamp"

	rows, err := conn.Query(context.Background(), sql, fromTimestamp, toTimestamp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Could not connect to database %v", err)))
		return
	}
	defer rows.Close()

	w.Header().Set("Transfer-Encoding", "chunked")

	w.Write([]byte("{\n"))

	previousMetricID := -1

	for rows.Next() {
		var metricID int
		var name string
		var timestamp time.Time
		var count float64
		var sum float64
		var sumSquares float64
		var min float64
		var max float64

		err = rows.Scan(&metricID, &name, &timestamp, &count, &sum, &sumSquares, &min, &max)
		if err != nil {
			log.Info("Could not scan row.")
			return
		}

		if metricID != previousMetricID {
			if previousMetricID != -1 {
				w.Write([]byte("],\n"))
			}
			w.Write([]byte("\"" + name + "\":[\n"))
		} else {
			w.Write([]byte(",\n"))
		}

		w.Write([]byte(fmt.Sprintf("{\"timestamp\":\"%s\",\"count\":%g,\"sum\":%g,\"sumSquares\":%g,\"min\":%g,\"max\":%g}",
			timestamp.Format(time.RFC3339), count, sum, sumSquares, min, max)))
		flusher.Flush()
		previousMetricID = metricID
	}

	if previousMetricID != -1 {
		w.Write([]byte("\n]"))
	}
	w.Write([]byte("\n}"))

	if rows.Err() != nil {
		log.Error("Something bad happenend while reading the rows:", rows.Err())
		return
	}
}

