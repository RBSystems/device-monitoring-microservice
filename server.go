package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/byuoitav/authmiddleware"
	"github.com/byuoitav/central-event-system/messenger"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/device-monitoring-microservice/handlers"
	"github.com/byuoitav/device-monitoring-microservice/monitoring"
	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/byuoitav/touchpanel-ui-microservice/socket"
	"github.com/fatih/color"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var addr string
var building string
var room string

func main() {
	//Our handy-dandy messenger to take our events to the hub
	m := messenger.BuildMessenger(os.Getenv("EVENT_ROUTER_ADDRESS"), Messenger, 1000)
	if _, exists := os.LookupEnv("SYSTEM_ROOM"); exists {
		//If the the we are working in a room, subscribe to that room and monitor it
		hostname := os.Getenv("SYSTEM_ID")
		building = strings.Split(hostname, "-")[0]
		room = strings.Split(hostname, "-")[1]
		r := make([]string)
		r = append(r, (building + "-" + room))
		m.SubscribeToRooms(r)
		go monitor(building, room, m)
	}

	// websocket
	hub := socket.NewHub(m)
	go WriteEventsToSocket(m, hub, statusinfrastructure.EventNodeStatus{})

	port := ":10000"
	router := echo.New()
	router.Pre(middleware.RemoveTrailingSlash())
	router.Use(middleware.CORS())
	// router.Use(echo.WrapMiddleware(authmiddleware.Authenticate))

	//	secure := router.Group("", echo.WrapMiddleware(authmiddleware.AuthenticateUser))
	secure := router.Group("", echo.WrapMiddleware(authmiddleware.Authenticate))

	// websocket
	router.GET("/websocket", func(context echo.Context) error {
		socket.ServeWebsocket(hub, context.Response().Writer, context.Request())
		return nil
	})

	secure.GET("/health", handlers.Health)
	secure.GET("/pulse", Pulse)
	secure.GET("/eventstatus", handlers.EventStatus, BindEventNode(en))
	secure.GET("/testevents", func(context echo.Context) error {
		//TODO - Confirm that this thing is a good translation
		//		en.Node.Write(messenger.Message{Header: events.TestStart, Body: []byte("test event")})
		e := events.Event{}
		e.AddToTags(events.TestStart)
		m.SendEvent(e)
		return nil
	})

	router.GET("/hostname", handlers.GetHostname)
	router.GET("/ip", handlers.GetIP)
	router.GET("/network", handlers.GetNetworkConnectedStatus)

	secure.GET("/reboot", handlers.RebootPi)

	secure.Static("/dash", "dash-dist")

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	router.StartServer(&server)

}

func Pulse(context echo.Context) error {
	err := monitoring.GetAndReportStatus(addr, building, room)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.JSON(http.StatusOK, "Pulse sent.")
}

func BindEventNode(en *events.EventNode) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(events.ContextEventNode, en)
			return next(c)
		}
	}
}

func monitor(building, room string, m *messenger.Messenger) {
	currentlyMonitoring := false

	for {
		shouldIMonitor := monitoring.ShouldIMonitorAPI()

		if shouldIMonitor && !currentlyMonitoring {
			color.Set(color.FgYellow, color.Bold)
			log.Printf("Starting monitoring of API")
			color.Unset()
			addr = monitoring.StartMonitoring(time.Second*300, "localhost:8000", building, room, m)
			currentlyMonitoring = true
		} else if currentlyMonitoring && shouldIMonitor {
		} else {
			color.Set(color.FgYellow, color.Bold)
			log.Printf("Stopping monitoring of API")
			color.Unset()

			// stop monitoring?
			monitoring.StopMonitoring()
			currentlyMonitoring = false
		}
		time.Sleep(time.Second * 15)
	}
}

func WriteEventsToSocket(m *messenger.Messenger, h *socket.Hub, t interface{}) {
	for {
		message := m.ReceiveEvent()

		if events.HasTag(message, TestExternal) {
			log.Printf(color.BlueString("Responding to external test event"))

			var s events.Event
			if len(os.Getenv("DEVELOPMENT_HOSTNAME")) > 0 {
				s.GeneratingSystem = os.Getenv("DEVELOPMENT_HOSTNAME")
			} else if len(os.Getenv("SYSTEM_ID")) > 0 {
				s.GeneratingSystem = os.Getenv("SYSTEM_ID")
			} else {
				s.GeneratingSystem, _ = os.Hostname()
			}
			s.AddToTags(events.TestExternal)
			m.SendEvent(s)
		}
		//I made it this far in this function (missing the others still)
		err := json.Unmarshal(message.Body, &t)
		if err != nil {
			log.Printf(color.RedString("failed to unmarshal message into Event type: %s", message.Body))
		} else {
			h.WriteToSockets(t)
		}
	}
}
