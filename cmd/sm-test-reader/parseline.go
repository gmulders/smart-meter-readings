package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	smr "github.com/gmulders/smart-meter-readings"
	log "github.com/sirupsen/logrus"
)

// parseNumber parses a string as an integer, assuming that the value has 3 decimals behind the decimal dot
func parseNumber(s string) (int64, error) {
	length := len(s)

	index := strings.Index(s, ".")
	// 1234.567 len = 8, index = 4, exponent = -3
	if length-index-1 != 3 {
		return 0, errors.New("expect value with a mantissa of length 3")
	}

	s2 := strings.Replace(s, ".", "", 1)
	value, err := strconv.ParseInt(s2, 10, 64)

	if err != nil {
		return 0, err
	}

	return value, nil
}

func valueRegex(unit string) string {
	return "\\((\\d+\\.\\d+)\\*" + unit + "\\)"
}

var timestampPattern = regexp.MustCompile(regexp.QuoteMeta("0-0:1.0.0") + "\\((\\d{12})(W|S)\\)")
var consumedTariff1Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:1.8.1") + valueRegex("kWh"))
var consumedTariff2Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:1.8.2") + valueRegex("kWh"))
var deliveredTariff1Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:2.8.1") + valueRegex("kWh"))
var deliveredTariff2Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:2.8.2") + valueRegex("kWh"))
var currentTariffPattern = regexp.MustCompile(regexp.QuoteMeta("0-0:96.14.0") + "\\((\\d+)\\)")
var powerConsumptionPattern = regexp.MustCompile(regexp.QuoteMeta("1-0:1.7.0") + valueRegex("kW"))
var powerDeliveryPattern = regexp.MustCompile(regexp.QuoteMeta("1-0:2.7.0") + valueRegex("kW"))
var powerConsumptionPhase1Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:21.7.0") + valueRegex("kW"))
var powerConsumptionPhase2Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:41.7.0") + valueRegex("kW"))
var powerConsumptionPhase3Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:61.7.0") + valueRegex("kW"))
var powerDeliveryPhase1Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:22.7.0") + valueRegex("kW"))
var powerDeliveryPhase2Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:42.7.0") + valueRegex("kW"))
var powerDeliveryPhase3Pattern = regexp.MustCompile(regexp.QuoteMeta("1-0:62.7.0") + valueRegex("kW"))

func parseLine(msg *smr.Telegram, line string) {
	log.Infof("line '%s'", line)

	matches := timestampPattern.FindStringSubmatch(line)
	if matches != nil {
		suffix := "+01:00"
		if matches[2] == "S" {
			suffix = "+02:00"
		}
		timestamp, err := time.ParseInLocation("060102150405Z07:00", matches[1]+suffix, time.Local)
		timestamp = timestamp.In(time.UTC)
		if err != nil {
			log.Error(err)
			return
		}
		msg.Timestamp = timestamp
		return
	}

	matches = consumedTariff1Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.ConsumedTariff1 = rat
		return
	}

	matches = consumedTariff2Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.ConsumedTariff2 = rat
		return
	}

	matches = deliveredTariff1Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.DeliveredTariff1 = rat
		return
	}

	matches = deliveredTariff2Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.DeliveredTariff2 = rat
		return
	}

	matches = currentTariffPattern.FindStringSubmatch(line)
	if matches != nil {
		i, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.CurrentTariff = int8(i)
		return
	}

	matches = powerConsumptionPattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerConsumption = rat
		return
	}

	matches = powerDeliveryPattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerDelivery = rat
		return
	}

	matches = powerConsumptionPhase1Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerConsumptionPhase1 = rat
		return
	}

	matches = powerConsumptionPhase2Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerConsumptionPhase2 = rat
		return
	}

	matches = powerConsumptionPhase3Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerConsumptionPhase3 = rat
		return
	}

	matches = powerDeliveryPhase1Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerDeliveryPhase1 = rat
		return
	}

	matches = powerDeliveryPhase2Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerDeliveryPhase2 = rat
		return
	}

	matches = powerDeliveryPhase3Pattern.FindStringSubmatch(line)
	if matches != nil {
		rat, err := parseNumber(matches[1])
		if err != nil {
			log.Error(err)
			return
		}
		msg.PowerDeliveryPhase3 = rat
		return
	}

	log.Infof("Unparsed line '%s'", line)
}
