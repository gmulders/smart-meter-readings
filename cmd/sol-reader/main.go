package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	smr "github.com/gmulders/smart-meter-readings"
	retry "github.com/sethvargo/go-retry"
	"github.com/simonvetter/modbus"
	log "github.com/sirupsen/logrus"
)

const (
	mqttTopicEnvName = "MQTT_TOPIC"
	modbusUrlEnvName = "MODBUS_URL"
)

func main() {

	channel := make(chan smr.SolarReadout)

	handler := smr.SolarReadoutHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := smr.BuildPahoClientConfig()

	mqttTopic := os.Getenv(mqttTopicEnvName)
	if mqttTopic == "" {
		log.Fatalf("Empty string %s '%s'", mqttTopicEnvName, mqttTopic)
	}

	//	modbusUrl := "tcp://192.168.1.127:1502"
	modbusUrl := os.Getenv(modbusUrlEnvName)
	if modbusUrl == "" {
		log.Fatalf("Empty string %s '%s'", modbusUrlEnvName, modbusUrl)
	}

	// Connect to the broker - this will return immediately after initiating the connection process
	cm, err := autopaho.NewConnection(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	go smr.WriteMeasurementStream[smr.SolarReadout](ctx, channel, handler, cm, mqttTopic)

	readSolarReadoutStream(modbusUrl, 1*time.Second, channel)
}

func readSolarReadoutStream(url string, timeout time.Duration, ch chan smr.SolarReadout) {
	measurement := &smr.SolarReadout{}

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     url,
		Timeout: timeout,
	})
	if err != nil {
		log.Fatal("could not create a new client", err)
	}

	strategy := retry.WithMaxRetries(8, retry.NewFibonacci(100*time.Millisecond))

	for {
		retry.Do(context.Background(), strategy, func(ctx context.Context) error {
			err := client.Open()
			if err != nil {
				log.Errorf("could not connect to the inverter: %v", err)
				return retry.RetryableError(err)
			}
			defer client.Close()
			measurement, err = ReadSolarMeasurement(client)
			if err != nil {
				log.Errorf("could not read the measurement: %v", err)
				return retry.RetryableError(err)
			}

			ch <- *measurement
			return nil
		})

		time.Sleep(10 * time.Second)
	}
}

func ReadSolarMeasurement(client *modbus.ModbusClient) (measurement *smr.SolarReadout, err error) {
	values, err := ReadModbusRegisterValues(client)
	if err != nil {
		return
	}

	for a, b := range *values {
		log.Infof("%s -> %v\n", a, b)
	}

	measurement = &smr.SolarReadout{}
	measurement.Timestamp = time.Now()
	measurement.Current = values.getScaledInt64("current", "current_scale", 3)
	measurement.L1Current = values.getScaledInt64("l1_current", "current_scale", 3)
	measurement.L1Voltage = values.getScaledInt64("l1_voltage", "voltage_scale", 3)
	measurement.L1NVoltage = values.getScaledInt64("l1n_voltage", "voltage_scale", 3)
	measurement.PowerAC = values.getScaledInt64("power_ac", "power_ac_scale", 0)
	measurement.Frequency = values.getScaledInt64("frequency", "frequency_scale", 3)
	measurement.PowerApparent = values.getScaledInt64("power_apparent", "power_apparent_scale", 0)
	measurement.PowerReactive = values.getScaledInt64("power_reactive", "power_reactive_scale", 0)
	measurement.PowerFactor = values.getScaledInt64("power_factor", "power_factor_scale", 2)
	measurement.EnergyTotal = values.getScaledInt64("energy_total", "energy_total_scale", 0)
	measurement.CurrentDC = values.getScaledInt64("current_dc", "current_dc_scale", 3)
	measurement.VoltageDC = values.getScaledInt64("voltage_dc", "voltage_dc_scale", 3)
	measurement.PowerDC = values.getScaledInt64("power_dc", "power_dc_scale", 0)
	measurement.Temperature = values.getScaledInt64("temperature", "temperature_scale", 2)

	return
}

type ModbusRegisterValue struct {
	Register ModbusRegister
	Value    interface{}
}

type ModbusRegisterValues map[string]ModbusRegisterValue

func scaleInt64(number int64, scale int16) int64 {
	if scale < 0 {
		if -scale > 19 { // 10 ** 20 overflows
			return 0
		}
		return number / int64(powUint64(10, uint64(-scale)))
	} else {
		if scale > 19 {
			return math.MaxInt64
		}
		return number * int64(powUint64(10, uint64(scale)))
	}
}

func powUint64(x, n uint64) uint64 {
	if n == 0 {
		return 1
	}
	if n == 1 {
		return x
	}
	y := powUint64(x, n/2)
	if n%2 == 0 {
		return y * y
	}
	return x * y * y
}

func (vs *ModbusRegisterValues) getScaledInt64(key string, scaleKey string, s int16) int64 {
	scale := s + (*vs)[scaleKey].Value.(int16)

	v := (*vs)[key]
	var int64Val int64
	if v.Register.DataType == INT16 {
		int64Val = int64(v.Value.(int16))
	} else if v.Register.DataType == UINT16 {
		int64Val = int64(v.Value.(uint16))
	} else if v.Register.DataType == ACC32 {
		int64Val = int64(v.Value.(uint32))
	} else {
		log.Fatal("Impossible data type conversion")
	}
	return scaleInt64(int64Val, scale)
}

func ReadModbusRegisterValues(client *modbus.ModbusClient) (*ModbusRegisterValues, error) {
	var readValues ModbusRegisterValues = ModbusRegisterValues{}

	for i := 1; i <= 3; i++ {
		registers := filter(Registers, func(r ModbusRegister) bool {
			return r.Batch == i
		})
		var max = maxBy(registers, func(a ModbusRegister, b ModbusRegister) int64 {
			return int64(a.Address) - int64(b.Address)
		})
		var min = maxBy(registers, func(a ModbusRegister, b ModbusRegister) int64 {
			return int64(b.Address) - int64(a.Address)
		})

		result, err := client.ReadRegisters(min.Address, max.Address+max.Size-min.Address, modbus.HOLDING_REGISTER)
		if err != nil {
			return nil, err
		}

		for _, r := range registers {
			readValues[r.Name] = ModbusRegisterValue{r, decodeValue(min.Address, result, r)}
		}
	}

	return &readValues, nil
}

func decodeValue(start uint16, result []uint16, r ModbusRegister) interface{} {
	slice := result[r.Address-start : r.Address-start+r.Size] // A register is 2 bytes
	switch r.DataType {
	case UINT16:
		return slice[0]
	case INT16:
		return int16(slice[0])
	case SCALE:
		return int16(slice[0])
	case ACC32:
		buf := new(bytes.Buffer)
		for _, v := range slice {
			binary.Write(buf, binary.BigEndian, v)
		}
		var value uint32
		binary.Read(buf, binary.BigEndian, &value)
		return value
	case STRING:
		buf := new(bytes.Buffer)
		for _, v := range slice {
			binary.Write(buf, binary.BigEndian, v)
		}
		bytes := buf.Bytes()
		count := len(bytes)
		for i, b := range bytes {
			if b == 0 {
				count = i
				break
			}
		}
		str := string(bytes[:count])
		return strings.TrimSpace(str)
	case FLOAT32:
		buf := new(bytes.Buffer)
		for _, v := range slice {
			binary.Write(buf, binary.LittleEndian, v)
		}
		var value float32
		binary.Read(buf, binary.LittleEndian, &value)
		return value
	case SUNSPEC_DID_INDEX:
		return SUNSPEC_DID_MAP[slice[0]]
	case INVERTER_STATUS_INDEX:
		return INVERTER_STATUS_MAP[slice[0]]
	}
	log.Fatal("Unknown type")
	return ""
}

func filter[T any](slice []T, f func(T) bool) []T {
	var n []T
	for _, e := range slice {
		if f(e) {
			n = append(n, e)
		}
	}
	return n
}

func maxBy[T any](l []T, comparator func(T, T) int64) T {
	var max = l[0]
	for _, e := range l {
		if comparator(max, e) < 0 {
			max = e
		}
	}
	return max
}

// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf

// Inverter Statuses
const (
	Ivs_I_STATUS_OFF           = 1
	Ivs_I_STATUS_SLEEPING      = 2
	Ivs_I_STATUS_STARTING      = 3
	Ivs_I_STATUS_MPPT          = 4
	Ivs_I_STATUS_THROTTLED     = 5
	Ivs_I_STATUS_SHUTTING_DOWN = 6
	Ivs_I_STATUS_FAULT         = 7
	Ivs_I_STATUS_STANDBY       = 8
)

type DataType int64

const (
	UINT16 DataType = iota
	SCALE
	INT16
	ACC32
	STRING
	FLOAT32
	SUNSPEC_DID_INDEX
	INVERTER_STATUS_INDEX
)

type ModbusRegister struct {
	Name        string
	Address     uint16
	Size        uint16
	DataType    DataType
	Description string
	Unit        string
	Batch       int
}

var SUNSPEC_DID_MAP = map[uint16]string{
	101: "Single Phase Inverter",
	102: "Split Phase Inverter",
	103: "Three Phase Inverter",
	201: "Single Phase Meter",
	202: "Split Phase Meter",
	203: "Wye 3P1N Three Phase Meter",
	204: "Delta 3P Three Phase Meter",
	802: "Battery",
	803: "Lithium Ion Bank Battery",
	804: "Lithium Ion String Battery",
	805: "Lithium Ion Module Battery",
	806: "Flow Battery",
	807: "Flow String Battery",
	808: "Flow Module Battery",
	809: "Flow Stack Battery",
}

var INVERTER_STATUS_MAP = []string{
	"Undefined",
	"Off",
	"Sleeping",
	"Grid Monitoring",
	"Producing",
	"Producing (Throttled)",
	"Shutting Down",
	"Fault",
	"Standby",
}

var Registers = []ModbusRegister{
	{"c_id", 0x9c40, 2, STRING, "SunSpec ID", "", 1},
	{"c_did", 0x9c42, 1, UINT16, "SunSpec DID", "", 1},
	{"c_length", 0x9c43, 1, UINT16, "SunSpec Length", "16Bit Words", 1},
	{"c_manufacturer", 0x9c44, 16, STRING, "Manufacturer", "", 1},
	{"c_model", 0x9c54, 16, STRING, "Model", "", 1},
	{"c_version", 0x9c6c, 8, STRING, "Version", "", 1},
	{"c_serialnumber", 0x9c74, 16, STRING, "Serial", "", 1},
	{"c_deviceaddress", 0x9c84, 1, UINT16, "Modbus ID", "", 1},

	{"c_sunspec_did", 0x9c85, 1, SUNSPEC_DID_INDEX, "SunSpec DID", "", 2},
	{"c_sunspec_length", 0x9c86, 1, UINT16, "Length", "16Bit Words", 2},

	{"current", 0x9c87, 1, UINT16, "Current", "A", 2},
	{"l1_current", 0x9c88, 1, UINT16, "L1 Current", "A", 2},
	{"l2_current", 0x9c89, 1, UINT16, "L2 Current", "A", 2},
	{"l3_current", 0x9c8a, 1, UINT16, "L3 Current", "A", 2},
	{"current_scale", 0x9c8b, 1, SCALE, "Current Scale Factor", "", 2},

	{"l1_voltage", 0x9c8c, 1, UINT16, "L1 Voltage", "V", 2},
	{"l2_voltage", 0x9c8d, 1, UINT16, "L2 Voltage", "V", 2},
	{"l3_voltage", 0x9c8e, 1, UINT16, "L3 Voltage", "V", 2},
	{"l1n_voltage", 0x9c8f, 1, UINT16, "L1-N Voltage", "V", 2},
	{"l2n_voltage", 0x9c90, 1, UINT16, "L2-N Voltage", "V", 2},
	{"l3n_voltage", 0x9c91, 1, UINT16, "L3-N Voltage", "V", 2},
	{"voltage_scale", 0x9c92, 1, SCALE, "Voltage Scale Factor", "", 2},

	{"power_ac", 0x9c93, 1, INT16, "Power", "W", 2},
	{"power_ac_scale", 0x9c94, 1, SCALE, "Power Scale Factor", "", 2},

	{"frequency", 0x9c95, 1, UINT16, "Frequency", "Hz", 2},
	{"frequency_scale", 0x9c96, 1, SCALE, "Frequency Scale Factor", "", 2},

	{"power_apparent", 0x9c97, 1, INT16, "Power (Apparent)", "VA", 2},
	{"power_apparent_scale", 0x9c98, 1, SCALE, "Power (Apparent) Scale Factor", "", 2},
	{"power_reactive", 0x9c99, 1, INT16, "Power (Reactive)", "VAr", 2},
	{"power_reactive_scale", 0x9c9a, 1, SCALE, "Power (Reactive) Scale Factor", "", 2},
	{"power_factor", 0x9c9b, 1, INT16, "Power Factor", "%", 2},
	{"power_factor_scale", 0x9c9c, 1, SCALE, "Power Factor Scale Factor", "", 2},

	{"energy_total", 0x9c9d, 2, ACC32, "Total Energy", "Wh", 2},
	{"energy_total_scale", 0x9c9f, 1, SCALE, "Total Energy Scale Factor", "", 2},

	{"current_dc", 0x9ca0, 1, UINT16, "DC Current", "A", 2},
	{"current_dc_scale", 0x9ca1, 1, SCALE, "DC Current Scale Factor", "", 2},

	{"voltage_dc", 0x9ca2, 1, UINT16, "DC Voltage", "V", 2},
	{"voltage_dc_scale", 0x9ca3, 1, SCALE, "DC Voltage Scale Factor", "", 2},

	{"power_dc", 0x9ca4, 1, INT16, "DC Power", "W", 2},
	{"power_dc_scale", 0x9ca5, 1, SCALE, "DC Power Scale Factor", "", 2},

	{"temperature", 0x9ca7, 1, INT16, "Temperature", "°C", 2},
	{"temperature_scale", 0x9caa, 1, SCALE, "Temperature Scale Factor", "", 2},

	{"status", 0x9cab, 1, INVERTER_STATUS_INDEX, "Status", "", 2},
	{"vendor_status", 0x9cac, 1, UINT16, "Vendor Status", "", 2},

	{"rrcr_state", 0xf000, 1, UINT16, "RRCR State", "", 3},
	{"active_power_limit", 0xf001, 1, UINT16, "Active Power Limit", "%", 3},
	{"cosphi", 0xf002, 2, FLOAT32, "CosPhi", "", 3},
}
