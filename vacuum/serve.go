package vacuum

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/johnDorian/rockbin/mqtt"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

func Serve(bin Bin, mqttClient mqtt.MqttConfig) {
	// on launch tell home assistant that we exist
	if err := mqttClient.SendConfig(); err != nil {
		log.WithError(err).Error("failed to send config to mqtt server")
	}

	// every minute send everything to the mqtt broker
	c := cron.New()
	_, err := c.AddFunc("@every 0h1m0s", func() {
		log.Debug("Running cron job")

		if err := mqttClient.SendConfig(); err != nil {
			log.WithError(err).Error("failed to send config to mqtt server")
		}

		bin.Update()

		if err := mqttClient.Send(bin.Value); err != nil {
			log.WithError(err).Error("failed to send value to mqtt server")
		}
	})
	if err != nil {
		log.WithError(err).Fatal("failed to setup cronjob")
	}

	c.Start()

	// Setup a file watcher to get instance updates on file changes
	log.Debug("Setting up file watcher for: ", bin.FilePath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				_ = event
				time.Sleep(time.Second * 1)

				bin.Update()
				log.Debug("Current bin value is:", bin.Value)
				if err := mqttClient.Send(bin.Value); err != nil {
					log.WithError(err).Error("failed to send value to mqtt server")
				}
			case err := <-watcher.Errors:
				log.WithError(err).Fatal("fatal error in file watcher")
			}
		}
	}()

	if err := watcher.Add(bin.FilePath); err != nil {
		log.Fatalln(err)
	}

	<-done

}
