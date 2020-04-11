package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	smr "github.com/gmulders/smart-meter-readings"
	"github.com/jackc/pgx/v4"
	nats "github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Could not connect to NATS: %v", err)
	}
	defer nc.Close()

	// postgres://YourUserName:YourPassword@YourHost:5432/YourDatabase
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Errorf("Unable to establish connection: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	nc.Subscribe("sm-telegram", func(msg *nats.Msg) {
		log.Info("Receiving smart meter telegram")
		telegram := smr.Telegram{}
		if err := json.Unmarshal(msg.Data, &telegram); err != nil {
			log.Errorf("Unable to unmarshal the telegram JSON: %v", err)
			return
		}

		saveTelegram(conn, &telegram)
	})

	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Error(err)
	}

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan // Blocks here until either SIGINT or SIGTERM is received.
}

func saveTelegram(conn *pgx.Conn, telegram *smr.Telegram) {
	sql := "insert into measurement (timestamp, value, name, dimensions) values " +
		"($1, $2, 'power-consumption', '{}'), " +
		"($1, $3, 'power-consumption-phase-1', '{}'), " +
		"($1, $4, 'power-consumption-phase-2', '{}'), " +
		"($1, $5, 'power-consumption-phase-3', '{}'), " +
		"($1, $6, 'power-consumed-tariff-1', '{}'), " +
		"($1, $7, 'power-consumed-tariff-2', '{}'), " +
		"($1, $8, 'power-delivery', '{}'), " +
		"($1, $9, 'power-delivery-phase-1', '{}'), " +
		"($1, $10, 'power-delivery-phase-2', '{}'), " +
		"($1, $11, 'power-delivery-phase-3', '{}'), " +
		"($1, $12, 'power-delivered-tariff-1', '{}'), " +
		"($1, $13, 'power-delivered-tariff-2', '{}')"

	_, err := conn.Exec(context.Background(), sql,
		telegram.Timestamp,
		telegram.PowerConsumption,
		telegram.PowerConsumptionPhase1,
		telegram.PowerConsumptionPhase2,
		telegram.PowerConsumptionPhase3,
		telegram.ConsumedTariff1,
		telegram.ConsumedTariff2,
		telegram.PowerDelivery,
		telegram.PowerDeliveryPhase1,
		telegram.PowerDeliveryPhase2,
		telegram.PowerDeliveryPhase3,
		telegram.DeliveredTariff1,
		telegram.DeliveredTariff2,
	)

	if err != nil {
		log.Errorf("Could not insert the measurements: %v", err)
	}
}
