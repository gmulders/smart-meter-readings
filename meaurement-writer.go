package meterstanden

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	log "github.com/sirupsen/logrus"
)

type IMeasurementHandler[M any] interface {
	GetTimestamp(m M) time.Time
	WriteMeasurement(writer io.Writer, m M, previous M) error
	ZeroMeasurement() M
}

func WriteMeasurementStream[M any](ctx context.Context, ch chan M, h IMeasurementHandler[M], cm *autopaho.ConnectionManager, topic string) {

	lastMonth := -1
	var writer *bufio.Writer
	var file *os.File
	var filename string
	var previousTelegram M

	for {
		telegram := <-ch

		json, err := json.Marshal(telegram)
		if err != nil {
			log.Error(err)
		}

		// AwaitConnection will return immediately if connection is up; adding this call stops publication whilst
		// connection is unavailable.
		asd, cancel := context.WithTimeout(ctx, 1*time.Second)
		err = cm.AwaitConnection(asd)
		cancel()
		if err != nil { // Should only happen when context is cancelled
			log.Errorf("Could not connect to the mqtt broker: %v", err)
			// We continue with saving the measurement, so that we don't lose data
		} else {
			publish := paho.Publish{
				QoS:     1,
				Topic:   topic,
				Payload: json,
			}
			_, err = cm.Publish(ctx, &publish)
			if err != nil {
				log.Errorf("Could not publish measurement event to mqtt: %v", err)
				// We continue with saving the measurement, so that we don't lose data
			}
		}

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

			filename, err = determineFilename(h.GetTimestamp(telegram))
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
