package main

import (
	"bufio"
	"context"
	"os"
	"strconv"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	smr "github.com/gmulders/smart-meter-readings"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

const (
	serialPortEnvName      = "SERIAL_PORT"
	influxServerUrlEnvName = "INLFUX_SERVER_URL"
	influxAuthTokenEnvName = "INLFUX_AUTH_TOKEN"
)

func main() {

	channel := make(chan smr.Telegram)
	handler := smr.TelegramHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serialPort := os.Getenv(serialPortEnvName)
	if serialPort == "" {
		log.Fatalf("Empty environment property %s '%s'", serialPortEnvName, serialPort)
	}

	config := &serial.Config{
		Name: serialPort,
		Baud: 115200,
	}

	influxServerUrl := os.Getenv(influxServerUrlEnvName)
	if influxServerUrl == "" {
		log.Fatalf("Empty environment property %s '%s'", influxServerUrlEnvName, influxServerUrl)
	}

	influxAuthToken := os.Getenv(influxAuthTokenEnvName)
	if influxAuthToken == "" {
		log.Fatalf("Empty environment property %s '%s'", influxAuthTokenEnvName, influxAuthToken)
	}

	client := influxdb2.NewClientWithOptions(
		influxServerUrl,
		influxAuthToken,
		influxdb2.DefaultOptions(),
	)

	serial, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(serial)

	go smr.WriteMeasurementStream[smr.Telegram](ctx, channel, handler, client)

	readTelegramStream(reader, channel)
}

func readTelegramStream(reader *bufio.Reader, ch chan smr.Telegram) {
	var crc uint16 = 0
	var telegram = &smr.Telegram{}

	for {
		// Read until next line feed (\n), this character is included in the resulting array
		bytes, err := reader.ReadBytes(0x0a)

		if err != nil {
			log.Error(err)
			continue
		}

		if len(bytes) == 0 {
			continue
		}

		// Instead of resetting the crc and telegram object here (if the first character is 0x2f), we do this at the
		// start of this function and after receiving the checksum of the telegram (whether it is valid or not).
		// if bytes[0] == 0x2f {
		//	crc = 0
		//	telegram = &smr.Telegram{}
		// }

		if bytes[0] != 0x21 {
			crc = crc16(crc, bytes)
			parseLine(telegram, string(bytes))
			continue
		}

		// When we get here, the line received contains a checksum and the telegram is finished

		crc = crc16(crc, []byte{0x21})
		expectedCRC, err := strconv.ParseUint(string(bytes[1:5]), 16, 16)
		if err != nil {
			log.Error(err)
			continue
		}

		if uint16(expectedCRC) != crc {
			log.Error("CRC mismatch")

			// Reset the crc and telegram object and continue, better luck next telegram
			crc = 0
			telegram = &smr.Telegram{}
			continue
		}

		// The telegram is valid; send it to the channel
		ch <- *telegram

		// Reset the crc and telegram object
		crc = 0
		telegram = &smr.Telegram{}
	}
}
