package meterstanden

import (
	"io"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Telegram represents a P1 telegram
type Telegram struct {
	Timestamp              time.Time `json:"time,omitempty"`
	ConsumedTariff1        int64     `json:"consumedTariff1,omitempty"`
	ConsumedTariff2        int64     `json:"consumedTariff2,omitempty"`
	DeliveredTariff1       int64     `json:"deliveredTariff1,omitempty"`
	DeliveredTariff2       int64     `json:"deliveredTariff2,omitempty"`
	CurrentTariff          int8      `json:"currentTariff,omitempty"`
	PowerConsumption       int64     `json:"powerConsumption,omitempty"`
	PowerDelivery          int64     `json:"powerDelivery,omitempty"`
	PowerConsumptionPhase1 int64     `json:"powerConsumptionPhase1,omitempty"`
	PowerConsumptionPhase2 int64     `json:"powerConsumptionPhase2,omitempty"`
	PowerConsumptionPhase3 int64     `json:"powerConsumptionPhase3,omitempty"`
	PowerDeliveryPhase1    int64     `json:"powerDeliveryPhase1,omitempty"`
	PowerDeliveryPhase2    int64     `json:"powerDeliveryPhase2,omitempty"`
	PowerDeliveryPhase3    int64     `json:"powerDeliveryPhase3,omitempty"`
}

type TelegramHandler struct {
	IMeasurementHandler[Telegram]
}

func (h TelegramHandler) CreatePoint(m Telegram) *write.Point {
	return influxdb2.NewPoint(
		"metering",
		map[string]string{
			"source": "p1-meter",
		},
		map[string]interface{}{
			"consumedTariff1":        float64(m.ConsumedTariff1),
			"consumedTariff2":        float64(m.ConsumedTariff2),
			"deliveredTariff1":       float64(m.DeliveredTariff1),
			"deliveredTariff2":       float64(m.DeliveredTariff2),
			"currentTarriff":         m.CurrentTariff,
			"powerConsumption":       float64(m.PowerConsumption),
			"powerDelivery":          float64(m.PowerDelivery),
			"powerConsumptionPhase1": float64(m.PowerConsumptionPhase1),
			"powerConsumptionPhase2": float64(m.PowerConsumptionPhase2),
			"powerConsumptionPhase3": float64(m.PowerConsumptionPhase3),
			"powerDeliveryPhase1":    float64(m.PowerDeliveryPhase1),
			"powerDeliveryPhase2":    float64(m.PowerDeliveryPhase2),
			"powerDeliveryPhase3":    float64(m.PowerDeliveryPhase3),
		},
		m.Timestamp,
	)
}

func (h TelegramHandler) GetTimestamp(t Telegram) time.Time {
	return t.Timestamp
}

func (h TelegramHandler) WriteMeasurement(writer io.Writer, telegram Telegram, previousTelegram Telegram) (err error) {
	if err = WriteValue(writer, telegram.Timestamp.Unix(), previousTelegram.Timestamp.Unix()); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.ConsumedTariff1, previousTelegram.ConsumedTariff1); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.ConsumedTariff2, previousTelegram.ConsumedTariff2); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.DeliveredTariff1, previousTelegram.DeliveredTariff1); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.DeliveredTariff2, previousTelegram.DeliveredTariff2); err != nil {
		return
	}
	if err = WriteValue(writer, int64(telegram.CurrentTariff), int64(previousTelegram.CurrentTariff)); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerConsumption, previousTelegram.PowerConsumption); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerDelivery, previousTelegram.PowerDelivery); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerConsumptionPhase1, previousTelegram.PowerConsumptionPhase1); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerConsumptionPhase2, previousTelegram.PowerConsumptionPhase2); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerConsumptionPhase3, previousTelegram.PowerConsumptionPhase3); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerDeliveryPhase1, previousTelegram.PowerDeliveryPhase1); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerDeliveryPhase2, previousTelegram.PowerDeliveryPhase2); err != nil {
		return
	}
	if err = WriteValue(writer, telegram.PowerDeliveryPhase3, previousTelegram.PowerDeliveryPhase3); err != nil {
		return
	}
	return nil
}

func (h TelegramHandler) ZeroMeasurement() Telegram {
	return zeroTelegram
}

var zeroTelegram = createZeroTelegram()

func createZeroTelegram() Telegram {
	telegram := Telegram{}
	telegram.Timestamp = time.Unix(0, 0)
	return telegram
}
