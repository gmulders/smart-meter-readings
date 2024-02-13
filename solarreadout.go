// Measurement represents a readout from the inverter
package meterstanden

import (
	"io"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type SolarReadout struct {
	Timestamp     time.Time `json:"time,omitempty"`
	Current       int64     `json:"current,omitempty"`       // mA
	L1Current     int64     `json:"l1Current,omitempty"`     // mA
	L1Voltage     int64     `json:"l1Voltage,omitempty"`     // mV
	L1NVoltage    int64     `json:"l1nVoltage,omitempty"`    // mV
	PowerAC       int64     `json:"powerAC,omitempty"`       // W
	Frequency     int64     `json:"frequency,omitempty"`     // mHz
	PowerApparent int64     `json:"powerApparent,omitempty"` // W
	PowerReactive int64     `json:"powerReactive,omitempty"` // W
	PowerFactor   int64     `json:"powerFactor,omitempty"`   // bp ?
	EnergyTotal   int64     `json:"energyTotal,omitempty"`   // Wh
	CurrentDC     int64     `json:"currentDC,omitempty"`     // mA
	VoltageDC     int64     `json:"voltageDC,omitempty"`     // mV
	PowerDC       int64     `json:"powerDC,omitempty"`       // W
	Temperature   int64     `json:"temperature,omitempty"`   // cC
}

type SolarReadoutHandler struct {
	IMeasurementHandler[SolarReadout]
}

func (h SolarReadoutHandler) CreatePoint(m SolarReadout) *write.Point {
	return influxdb2.NewPoint(
		"solar",
		map[string]string{
			"source": "solar-edge-1",
		},
		map[string]interface{}{
			"current":       float64(m.Current) / 1000.0,
			"l1current":     float64(m.L1Current) / 1000.0,
			"l1voltage":     float64(m.L1Voltage) / 1000.0,
			"l1nvoltage":    float64(m.L1NVoltage) / 1000.0,
			"powerAC":       float64(m.PowerAC),
			"frequency":     float64(m.Frequency) / 1000.0,
			"powerApparent": float64(m.PowerApparent),
			"powerReactive": float64(m.PowerReactive),
			"powerFactor":   float64(m.PowerFactor) / 10000.0,
			"energyTotal":   float64(m.EnergyTotal),
			"currentDC":     float64(m.CurrentDC) / 1000.0,
			"voltageDC":     float64(m.VoltageDC) / 1000.0,
			"powerDC":       float64(m.PowerDC),
			"temperature":   float64(m.Temperature) / 100.0,
		},
		m.Timestamp,
	)
}

func (h SolarReadoutHandler) GetTimestamp(t SolarReadout) time.Time {
	return t.Timestamp
}

func (h SolarReadoutHandler) WriteMeasurement(writer io.Writer, s SolarReadout, previous SolarReadout) error {
	if err := WriteValue(writer, s.Timestamp.Unix(), previous.Timestamp.Unix()); err != nil {
		return err
	}
	if err := WriteValue(writer, s.Current, previous.Current); err != nil {
		return err
	}
	if err := WriteValue(writer, s.L1Current, previous.L1Current); err != nil {
		return err
	}
	if err := WriteValue(writer, s.L1Voltage, previous.L1Voltage); err != nil {
		return err
	}
	if err := WriteValue(writer, s.L1NVoltage, previous.L1NVoltage); err != nil {
		return err
	}
	if err := WriteValue(writer, s.PowerAC, previous.PowerAC); err != nil {
		return err
	}
	if err := WriteValue(writer, s.Frequency, previous.Frequency); err != nil {
		return err
	}
	if err := WriteValue(writer, s.PowerApparent, previous.PowerApparent); err != nil {
		return err
	}
	if err := WriteValue(writer, s.PowerReactive, previous.PowerReactive); err != nil {
		return err
	}
	if err := WriteValue(writer, s.PowerFactor, previous.PowerFactor); err != nil {
		return err
	}
	if err := WriteValue(writer, s.EnergyTotal, previous.EnergyTotal); err != nil {
		return err
	}
	if err := WriteValue(writer, s.CurrentDC, previous.CurrentDC); err != nil {
		return err
	}
	if err := WriteValue(writer, s.VoltageDC, previous.VoltageDC); err != nil {
		return err
	}
	if err := WriteValue(writer, s.PowerDC, previous.PowerDC); err != nil {
		return err
	}
	if err := WriteValue(writer, s.Temperature, previous.Temperature); err != nil {
		return err
	}
	return nil
}

func (h SolarReadoutHandler) ZeroMeasurement() SolarReadout {
	return zeroSolarReadout
}

var zeroSolarReadout = createZeroSolarReadout()

func createZeroSolarReadout() SolarReadout {
	solarReadout := SolarReadout{}
	solarReadout.Timestamp = time.Unix(0, 0)
	return solarReadout
}
