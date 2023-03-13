package meterstanden

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	log "github.com/sirupsen/logrus"
)

const (
	mqttBrokerUrlEnvName           = "MQTT_BROKER_URL"
	mqttConnectionKeepAliveEnvName = "MQTT_CONNECTION_KEEP_ALIVE"
	mqttClientIdEnvName            = "MQTT_CLIENT_ID"
)

func BuildPahoClientConfig() autopaho.ClientConfig {
	mqttBrokerUrlStr := os.Getenv(mqttBrokerUrlEnvName)
	mqttBrokerUrl, err := url.Parse(mqttBrokerUrlStr)
	if err != nil {
		log.Panicf("Could not parse %s '%s'", mqttBrokerUrlEnvName, mqttBrokerUrlStr)
	}

	mqttConnectionKeepAliveStr := os.Getenv(mqttConnectionKeepAliveEnvName)
	mqttConnectionKeepAlive, err := strconv.Atoi(mqttConnectionKeepAliveStr)
	if err != nil {
		log.Panicf("Could not parse %s '%s'", mqttConnectionKeepAliveEnvName, mqttConnectionKeepAliveStr)
	}

	mqttClientId := os.Getenv(mqttClientIdEnvName)
	if mqttClientId == "" {
		log.Panicf("Empty string %s '%s'", mqttClientIdEnvName, mqttClientId)
	}

	return autopaho.ClientConfig{
		BrokerUrls:        []*url.URL{mqttBrokerUrl},
		KeepAlive:         uint16(mqttConnectionKeepAlive),
		ConnectRetryDelay: 1 * time.Second,
		OnConnectionUp:    func(*autopaho.ConnectionManager, *paho.Connack) { log.Info("connected to mqtt") },
		OnConnectError:    func(err error) { log.Infof("could not connect to mqtt: %s\n", err) },
		Debug:             paho.NOOPLogger{},
		ClientConfig: paho.ClientConfig{
			ClientID:      mqttClientId,
			OnClientError: func(err error) { fmt.Printf("server requested disconnect: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	}
}
