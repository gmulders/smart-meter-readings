package meterstanden

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	log "github.com/sirupsen/logrus"
)

type IMeasurementHandler[M any] interface {
	GetTimestamp(m M) time.Time
	CreatePoint(m M) *write.Point
	WriteMeasurement(writer io.Writer, m M, previous M) error
	ZeroMeasurement() M
}

func WriteMeasurementStream[M any](ctx context.Context, ch chan M, h IMeasurementHandler[M], client influxdb2.Client) {

	lastMonth := -1
	var writer *bufio.Writer
	var file *os.File
	var filename string
	var previousTelegram M

	writeAPI := client.WriteAPI("ha", "electricity")

	for {
		telegram := <-ch

		writeAPI.WritePoint(h.CreatePoint(telegram))

		currentMonth := int(h.GetTimestamp(telegram).Month())

		if currentMonth != lastMonth {
			lastMonth = currentMonth
			if file != nil {
				log.Info("Closing the file")
				writer.Flush()
				file.Close()

				// Start a go routine to zip the old file
				go gzipFile(filename)
			}

			filename, err := determineFilename(h.GetTimestamp(telegram))
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
			previousTelegram = h.ZeroMeasurement()
		}

		h.WriteMeasurement(writer, telegram, previousTelegram)

		// Flush the data to the file. This is relatively expensive since we only write a couple of bytes,
		// however we don't lose data this way.
		writer.Flush()

		previousTelegram = telegram
	}
}

func determineFilename(ts time.Time) (string, error) {
	// Return the first name for which no file exists
	for i := 1; i <= 999; i++ {
		filename := fmt.Sprintf("data/%d-%02d.%03d.bin", ts.Year(), ts.Month(), i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return filename, nil
		}
	}
	return "", errors.New("could not determine filename")
}

func WriteValue(writer io.Writer, newValue int64, oldValue int64) error {
	buff := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buff, newValue-oldValue)
	_, err := writer.Write(buff[:n])
	return err
}
