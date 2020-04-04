package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	m "github.com/gmulders/meterstanden"
	nats "github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Expecting name of usb device")
	}

	config := &serial.Config{
		Name: os.Args[1],
		Baud: 115200,
	}

	serial, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(serial)

	// Connect to a server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Errorf("Could not connect to NATS: %v", err)
		return
	}
	defer nc.Close()

	channel := make(chan m.Telegram)

	go writeTelegramStream(channel, nc)

	readTelegramStream(reader, channel)
}

var zeroTelegram = createZeroTelegram()

func createZeroTelegram() m.Telegram {
	telegram := m.Telegram{}
	telegram.Timestamp = time.Unix(0, 0)
	return telegram
}

func writeTelegramStream(ch chan m.Telegram, nc *nats.Conn) {

	lastMonth := -1
	var writer *bufio.Writer
	var file *os.File
	var filename string
	var previousTelegram m.Telegram

	for {
		telegram := <-ch

		json, err := json.MarshalIndent(telegram, "", "\t")
		if err != nil {
			log.Error(err)
		}

		log.Info(string(json))

		err = nc.Publish("p1-telegram", json)
		if err != nil {
			log.Errorf("Could not publish 'p1-telegram' event to NATS: %v", err)
			// We continue with saving the telegram, so that we don't lose data
		}

		currentMonth := int(telegram.Timestamp.Month())

		if currentMonth != lastMonth {
			lastMonth = currentMonth
			if file != nil {
				log.Info("Closing the file")
				writer.Flush()
				file.Close()

				// Start a go routine to zip the old file
				go gzipFile(filename)
			}
			ts := telegram.Timestamp
			filename = fmt.Sprintf("meterstanden-%d-%02d.bin", ts.Year(), ts.Month())
			newFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)

			if err != nil {
				log.Fatal(err)
			}

			file = newFile
			// Create a buffering writer
			writer = bufio.NewWriter(file)

			// We start a new file, but we need to have a telegram to substract from the current telegram,
			// so we take a telegram with only zero values.
			previousTelegram = zeroTelegram
		}

		writeTelegram(writer, telegram, previousTelegram)

		// Flush the data to the file. This is relatively expensive since we only write a couple of bytes,
		// however we don't lose data this way.
		writer.Flush()

		previousTelegram = telegram
	}
}

func writeTelegram(writer io.Writer, telegram m.Telegram, previousTelegram m.Telegram) {
	writeValue(writer, telegram.Timestamp.Unix(), previousTelegram.Timestamp.Unix())
	writeValue(writer, telegram.ConsumedTariff1, previousTelegram.ConsumedTariff1)
	writeValue(writer, telegram.ConsumedTariff2, previousTelegram.ConsumedTariff2)
	writeValue(writer, telegram.DeliveredTariff1, previousTelegram.DeliveredTariff1)
	writeValue(writer, telegram.DeliveredTariff2, previousTelegram.DeliveredTariff2)
	writeValue(writer, int64(telegram.CurrentTariff), int64(previousTelegram.CurrentTariff))
	writeValue(writer, telegram.PowerConsumption, previousTelegram.PowerConsumption)
	writeValue(writer, telegram.PowerDelivery, previousTelegram.PowerDelivery)
	writeValue(writer, telegram.PowerConsumptionPhase1, previousTelegram.PowerConsumptionPhase1)
	writeValue(writer, telegram.PowerConsumptionPhase2, previousTelegram.PowerConsumptionPhase2)
	writeValue(writer, telegram.PowerConsumptionPhase3, previousTelegram.PowerConsumptionPhase3)
	writeValue(writer, telegram.PowerDeliveryPhase1, previousTelegram.PowerDeliveryPhase1)
	writeValue(writer, telegram.PowerDeliveryPhase2, previousTelegram.PowerDeliveryPhase2)
	writeValue(writer, telegram.PowerDeliveryPhase3, previousTelegram.PowerDeliveryPhase3)
}

func writeValue(writer io.Writer, newValue int64, oldValue int64) {
	buff := make([]byte, binary.MaxVarintLen64)
	fmt.Printf("value: %d\n", newValue-oldValue)
	n := binary.PutVarint(buff, newValue-oldValue)
	if _, err := writer.Write(buff[:n]); err != nil {
		log.Fatal(err)
	}
}

func readTelegramStream(reader *bufio.Reader, ch chan m.Telegram) {
	var crc uint16
	var telegram *m.Telegram

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

		if bytes[0] == 0x2f {
			crc = 0
			telegram = &m.Telegram{}
		}

		if bytes[0] != 0x21 {
			crc = crc16(crc, bytes)

			parseLine(telegram, string(bytes))

			continue
		}

		crc = crc16(crc, []byte{0x21})
		expectedCRC, err := strconv.ParseUint(string(bytes[1:5]), 16, 16)
		if err != nil {
			log.Error(err)
			continue
		}

		if uint16(expectedCRC) != crc {
			log.Error("CRC mismatch")
		}

		// Handle the telegram
		ch <- *telegram
	}
}
