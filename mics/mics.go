package mics

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/av-api/status"
	ei "github.com/byuoitav/event-router-microservice/eventinfrastructure/event"
)

var ticker time.Ticker
const STATUS_OK

func GetMicBatteries(interval time.Duration, shureAddr, format, building, room string) error {

	//start by getting the Shure device from the database
	shure, err := dbo.GetDevicesByBuildingAndRoomAndRole(building, room, "Receiver")
	if err != nil {
		errorMessage := "Could not find Shure device in room " + err.Error()
		log.Printf(errorMessage)
		return errors.New(errorMessage)
	}

	//validate that there is only one shure in a room
	if len(shure) != 1 {
		errorMessage := "Invalid Shure receiver configuration detected"
		log.Printf(errorMessage)
		return errors.New(errorMessage)
	}

	ticker := time.NewTicker(interval)

	go func() {
		for _ = range ticker.C {
			//for each configured port query mic status and publish result
			for _, port := range shure[0].Ports {

				response, err := QueryMicBattery(port.Name, shureAddr)
				if err != nil {
					//TODO publish error here
					SendEvent(ei.ERROR,
					ei.AUTOGENERATED,
					port.Source,
					room,
					building,
					"battery",
					"not responding",


				}
				//TODO publish result here

			}
		}
	}()

}

func QueryMicBattery(address, channel, format string) (status.Battery, error) {

	//build address
	address := fmt.Sprintf("http://%s/%s/battery/%s", address, channel, format)

	//send request
	response, err := http.Get(address)
	if err != nil {
		return status.Battery{}, err
	} else if response.StatusCode != STATUS_OK {
		return statusBattery{}, errors.New("Non-200 response from shure audio microservice")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return status.Battery{}, err
	}

	var battery status.Battery
	err = json.Unmarshal(body, &battery)
	if err != nil {
		return status.Battery{}, err
	}

	return battery, nil

}
