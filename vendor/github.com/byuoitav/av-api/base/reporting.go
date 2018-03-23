package base

import (
	"os"
	"strings"
	"time"

	"github.com/byuoitav/event-router-microservice/eventinfrastructure"
)

var EventNode *eventinfrastructure.EventNode

func PublishHealth(e eventinfrastructure.Event) {
	Publish(e, false)
}

func Publish(e eventinfrastructure.Event, Error bool) error {
	var err error

	e.Timestamp = time.Now().Format(time.RFC3339)
	if len(os.Getenv("LOCAL_ENVIRONMENT")) > 0 {
		e.Hostname = os.Getenv("PI_HOSTNAME")
		if len(os.Getenv("DEVELOPMENT_HOSTNAME")) > 0 {
			e.Hostname = os.Getenv("DEVELOPMENT_HOSTNAME")
		}
	} else {
		// isn't it running in a docker container in aws? this won't work?
		e.Hostname, err = os.Hostname()
	}
	if err != nil {
		return err
	}

	e.LocalEnvironment = len(os.Getenv("LOCAL_ENVIRONMENT")) > 0

	if !Error {
		EventNode.PublishEvent(e, eventinfrastructure.APISuccess)
	} else {
		EventNode.PublishEvent(e, eventinfrastructure.APIError)
	}

	return err
}

func SendEvent(Type eventinfrastructure.EventType,
	Cause eventinfrastructure.EventCause,
	Device string,
	Room string,
	Building string,
	InfoKey string,
	InfoValue string,
	Requestor string,
	Error bool) error {

	e := eventinfrastructure.EventInfo{
		Type:           Type,
		EventCause:     Cause,
		Device:         Device,
		EventInfoKey:   InfoKey,
		EventInfoValue: InfoValue,
		Requestor:      Requestor,
	}

	err := Publish(eventinfrastructure.Event{
		Event:    e,
		Building: Building,
		Room:     Room,
	}, Error)

	return err
}

func PublishError(errorStr string, cause eventinfrastructure.EventCause) {
	e := eventinfrastructure.EventInfo{
		Type:           eventinfrastructure.ERROR,
		EventCause:     cause,
		EventInfoKey:   "Error String",
		EventInfoValue: errorStr,
	}

	building := ""
	room := ""

	if len(os.Getenv("LOCAL_ENVIRONMENT")) > 0 {
		if len(os.Getenv("PI_HOSTNAME")) > 0 {
			name := os.Getenv("PI_HOSTNAME")
			roomInfo := strings.Split(name, "-")
			building = roomInfo[0]
			room = roomInfo[1]
			e.Device = roomInfo[2]
		}
	}

	Publish(eventinfrastructure.Event{
		Event:    e,
		Building: building,
		Room:     room,
	}, true)
}
