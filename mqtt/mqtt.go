package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttConfig struct {
	Name              string      `json:"name"`
	UnitOfMeasurement string      `json:"unit_of_measurement"`
	StateTopic        string      `json:"state_topic"`
	UniqueID          string      `json:"unique_id"`
	CAPath            string      `json:"-"`
	KeyPath           string      `json:"-"`
	CertPath          string      `json:"-"`
	MaxConnectionTime int         `json:"-"`
	Client            mqtt.Client `json:"-"`
	ConfigTopic       string      `json:"-"`
	Server            string      `json:"-"`
	Username          string      `json:"-"`
	Password          string      `json:"-"`
}

func (m *MqttConfig) ConnectWithBackoff() error {
	connectionBackoff := backoff.NewExponentialBackOff()
	connectionBackoff.InitialInterval = 1 * time.Second
	connectionBackoff.MaxElapsedTime = time.Duration(m.MaxConnectionTime) * time.Second
	err := backoff.RetryNotifyWithTimer(m.Connect,
		connectionBackoff,
		func(e error, d time.Duration) {
			log.WithField("backoff", d.String()).
				WithError(e).
				Debug("mqtt connection attempt failed")
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("mqtt connection to: %s failed after trying for %s seconds", m.Server, connectionBackoff.MaxElapsedTime)
	}
	return nil
}

func (m *MqttConfig) Connect() error {
	mqttURL, err := url.Parse(m.Server)
	if err != nil {
		return err
	}

	if mqttURL.Scheme == "mqtt" {
		mqttURL.Scheme = "tcp"
	}

	if mqttURL.Scheme == "mqtts" {
		mqttURL.Scheme = "ssl"
	}

	url := mqttURL.String()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)

	opts.SetClientID(m.UniqueID)
	if len(m.Username) > 0 {
		opts.SetUsername(m.Username)
	}
	if len(m.Password) > 0 {
		opts.SetPassword(m.Password)
	}

	rootCerts, err := m.getCACertificates()
	if err != nil {
		return err
	}

	clientCertificates, err := m.getClientCertificates()
	if err != nil {
		return err
	}

	opts.SetTLSConfig(&tls.Config{
		Certificates: clientCertificates,
		RootCAs:      rootCerts,
	})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	for !token.WaitTimeout(2 * time.Second) {
	}

	if err := token.Error(); err != nil {
		return err
	}
	if client.IsConnected() {
		log.WithFields(log.Fields{"mqtt_broker": url}).Info("Connected to mqtt broker")
	}

	m.Client = client
	return nil
}

// SendConfig send the home assistant auto discovery config to mqtt
func (m *MqttConfig) SendConfig() error {
	mqttPayload, err := preparePayload(m)
	if err != nil {
		return err
	}
	log.Debug("Sending mqtt config")
	err = sendMessage(m.Client, m.ConfigTopic, mqttPayload, true)
	return err
}

// Send any data to home assistant
func (m *MqttConfig) Send(data string) error {
	log.Debug("Sending mqtt message")
	err := sendMessage(m.Client, m.StateTopic, data, false)
	return err
}

func sendMessage(client mqtt.Client, topic string, data string, retain bool) error {
	token := client.Publish(topic, 0, retain, data)
	if token.Error() != nil {
		log.Fatalln(token.Error())
		return token.Error()
	}
	return nil
}

func preparePayload(data interface{}) (string, error) {
	mqttPayload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(mqttPayload), nil
}

func (m *MqttConfig) getCACertificates() (*x509.CertPool, error) {
	if m.CAPath == "" {
		return x509.SystemCertPool()
	}

	certs := x509.NewCertPool()

	file, err := os.Open(m.CAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ca certificate: %w", err)
	}

	pem, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read ca certificate: %w", err)
	}

	if ok := certs.AppendCertsFromPEM(pem); !ok {
		return nil, errors.New("failed to parse ca certificate: not a valid PEM file")
	}

	return certs, nil
}

func (m *MqttConfig) getClientCertificates() ([]tls.Certificate, error) {
	if m.CertPath == "" && m.KeyPath == "" {
		return []tls.Certificate{}, nil
	}

	clientCert, err := tls.LoadX509KeyPair(m.CertPath, m.KeyPath)
	if err != nil {
		return []tls.Certificate{}, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return []tls.Certificate{clientCert}, nil
}
