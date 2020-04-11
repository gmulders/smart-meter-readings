package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	smr "github.com/gmulders/smart-meter-readings"
	nats "github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
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

	channel := make(chan smr.Telegram)

	go writeTelegramStream(channel, nc)

	readTelegramStream(reader, channel)
}

var zeroTelegram = createZeroTelegram()

func createZeroTelegram() smr.Telegram {
	telegram := smr.Telegram{}
	telegram.Timestamp = time.Unix(0, 0)
	return telegram
}

func determineFilename(ts time.Time) (string, error) {
	// Return the first name for which no file exists
	for i := 1; i <= 999; i++ {
		filename := fmt.Sprintf("meterstanden-%d-%02d.%03d.bin", ts.Year(), ts.Month(), i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return filename, nil
		}
	}
	return "", errors.New("Could not determine filename")
}

func writeTelegramStream(ch chan smr.Telegram, nc *nats.Conn) {

	lastMonth := -1
	var writer *bufio.Writer
	var file *os.File
	var filename string
	var previousTelegram smr.Telegram

	for {
		telegram := <-ch

		json, err := json.MarshalIndent(telegram, "", "\t")
		if err != nil {
			log.Error(err)
		}

		err = nc.Publish("sm-telegram", json)
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

			filename, err = determineFilename(telegram.Timestamp)
			if err != nil {
				log.Fatal(err)
			}

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

func writeTelegram(writer io.Writer, telegram smr.Telegram, previousTelegram smr.Telegram) {
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
	n := binary.PutVarint(buff, newValue-oldValue)
	if _, err := writer.Write(buff[:n]); err != nil {
		log.Fatal(err)
	}
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
