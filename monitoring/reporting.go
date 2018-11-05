package monitoring

import (
	"os"
	"strings"
	"time"

	"github.com/byuoitav/common/v2/events"
)

func SendEvent(Type events.EventType,
	Cause events.EventCause,
	Device string,
	DeviceID string,
	Room string,
	Building string,
	InfoKey string,
	InfoValue string,
	Error bool) error {

	e := events.EventInfo{
		Type:           Type,
		EventCause:     Cause,
		Device:         Device,
		DeviceID:       DeviceID,
		EventInfoKey:   InfoKey,
		EventInfoValue: InfoValue,
	}

	err := Publish(events.Event{
		Event:    e,
		Building: Building,
		Room:     Room,
	}, Error)

	return err
}

func PublishError(errorStr string, cause events.EventCause) {
	e := events.Event{
		Key:   "Error String",
		Value: errorStr,
	}
	e.AddToTags(cause)
	e.AddToTags(events.Error)

	building := ""
	room := ""

	if len(os.Getenv("ROOM_SYSTEM")) > 0 {
		if len(os.Getenv("SYSTEM_ID")) > 0 {
			name := os.Getenv("SYSTEM_ID")
			roomInfo := strings.Split(name, "-")
			building = roomInfo[0]
			room = roomInfo[1]
			e.TargetDevice = events.GenerateBasicDeviceInfo(name)
			e.GeneratingSystem = name
			e.AffectedRoom = events.GenerateBasicRoomInfo(building + "-" + room)
		}
	}
	Publish(e)
}

func Publish(e events.Event) error {
	var err error

	// create the event
	e.Timestamp = time.Now()
	if len(os.Getenv("ROOM_SYSTEM")) > 0 {
		e.GeneratingSystem = os.Getenv("SYSTEM_ID")
		if len(os.Getenv("DEVELOPMENT_HOSTNAME")) > 0 {
			e.GeneratinSystem = os.Getenv("DEVELOPMENT_HOSTNAME")
		}
	} else {
		e.GeneratingSystem, err = os.Hostname()
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	if len(os.Getenv("ROOM_SYSTEM")) > 0 {
		e.AddToTags(events.RoomSystem)
	}

	m.SendEvent(e)
	return err
}
