package microservice

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/byuoitav/device-monitoring-microservice/logstash"
)

func CheckMicroservices() {

	log.Printf("Checking health of microservices...")

	hostname := os.Getenv("PI_HOSTNAME")
	splitHostname := strings.Split(hostname, "-")
	building := splitHostname[0]
	room := splitHostname[1]

	for service, port := range microservices {

		log.Printf("Checking %s on port %v", service, port)

		portString := strconv.Itoa(port)
		address := "localhost:" + portString + "/health"
		response, err := http.Get(address)
		if err != nil {

			message := "Microservice: " + service + " not responding"
			log.Printf(message)

			timestamp := string(time.Now().Format(time.RFC3339))
			logstash.SendEvent(building, room, timestamp, service, "not responding", "server")

		}

		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)

		log.Printf("Response: %s", body)

	}

}

var microservices = map[string]int{
	"event-router-microservice":           7000,
	"salt-event-proxy":                    7010,
	"av-api":                              8000,
	"pjlink-microservice":                 8005,
	"configuration-database-microservice": 8006,
	"sony-control-microservice":           8007,
	"london-audio-microservice":           8009,
	"pulse-eight-neo-microservice":        8011,
	"adcp-control-microservice":           8012,
	"touchpanel-ui-microservice":          8888,
}
