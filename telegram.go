package meterstanden

import (
	"time"
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
