package mqtt

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		username string
		password string
	}{
		{"user", "pass"},
		{"user1", "!$%&/()?#+*12345"},
		{"user2", `hello"world`},
	}
	resource, pool := spinUpMQTT()
	defer func() {
		if err := pool.Purge(resource); err != nil {
			_, _ = fmt.Printf("WARNING: failed to cleanup mqtt server: %v\n", err)
		}
	}()

	for _, up := range testData {
		config := MqttConfig{Name: "hello",
			UnitOfMeasurement: "hello",
			StateTopic:        "hello",
			ConfigTopic:       "hello",
			UniqueID:          "hello",
			Server:            fmt.Sprintf("mqtts://localhost:%v", resource.GetPort("1883/tcp")),
			Username:          up.username,
			Password:          up.password,
			CAPath:            "../tests/certificates/rootCA.pem",
			CertPath:          "../tests/certificates/server.crt",
			KeyPath:           "../tests/certificates/server.key",
		}

		err := config.Connect()
		require.NoError(t, err, "require MQTT connection")
		assert.True(config.Client.IsConnected())
	}
}

func TestPreparePayload(t *testing.T) {
	assert := assert.New(t)
	var TestData = []struct {
		item1       string
		item2       string
		expected    string
		expectError bool
	}{
		{"item1", "item2", `{"item1":"item1","item2":"item2"}`, false},
	}

	for _, test := range TestData {
		data := struct {
			Item1 string `json:"item1"`
			Item2 string `json:"item2"`
		}{
			test.item1, test.item2,
		}
		payload, err := preparePayload(data)
		assert.Equal(payload, test.expected)
		if test.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}

}

func spinUpMQTT() (*dockertest.Resource, *dockertest.Pool) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	dir, _ := os.Getwd()
	options := &dockertest.RunOptions{
		Repository:   "eclipse-mosquitto",
		Tag:          "2.0.9",
		Name:         "mosquitto",
		ExposedPorts: []string{"1883", "9001"},
		Mounts: []string{
			fmt.Sprintf("%v:/mosquitto/config/mosquitto.conf", path.Join(dir, "../tests/mosquitto.conf")),
			fmt.Sprintf("%v:/certs", path.Join(dir, "../tests/certificates")),
			fmt.Sprintf("%v:/password.txt", path.Join(dir, "../tests/password.txt")),
		},
	}

	resource, err := pool.RunWithOptions(options)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	return resource, pool
}
