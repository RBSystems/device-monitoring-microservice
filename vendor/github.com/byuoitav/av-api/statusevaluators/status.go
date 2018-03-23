package statusevaluators

import (
	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/configuration-database-microservice/structs"
)

type PowerStatus struct {
	Power string `json:"power"`
}

type BlankedStatus struct {
	Blanked bool `json:"blanked"`
}

type MuteStatus struct {
	Muted bool `json:"muted"`
}

type Input struct {
	Input string `json:"input,omitempty"`
}

type AudioList struct {
	Inputs []Input `json"inputs"`
}

type VideoList struct {
	Inputs []Input `json:"inputs"`
}

type Volume struct {
	Volume int `json:"volume"`
}

type Battery struct {
	Battery int `json:"battery"`
}

//represents output from a device, use Error field to flag errors
type Status struct {
	Status            map[string]interface{} `json:"status"`
	DestinationDevice base.DestinationDevice `json:"destination_device"`
}

//represents a status response, including the generator that created the command that returned the status
type StatusResponse struct {
	SourceDevice      structs.Device         `json:"source_device"`
	DestinationDevice base.DestinationDevice `json:"destination_device"`
	Callback          func(base.StatusPackage, chan<- base.StatusPackage) error
	Generator         string                 `json:"generator"`
	Status            map[string]interface{} `json:"status"`
	ErrorMessage      *string                `json:"error"`
}

//StatusCommand contains information to issue a status command against a device
type StatusCommand struct {
	Action            structs.Command `json:"action"`
	Device            structs.Device  `json:"device"`
	Callback          func(base.StatusPackage, chan<- base.StatusPackage) error
	Generator         string                 `json:"generator"`
	DestinationDevice base.DestinationDevice `json:"destination"`
	Parameters        map[string]string      `json:"parameters"`
}

//DestinationDevice represents the device whose status is being queried by user

const FLAG = "STATUS"
