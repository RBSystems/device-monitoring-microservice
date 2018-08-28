package jobs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/byuoitav/common/events"
	"github.com/byuoitav/common/log"
)

var (
	runners   []*runner
	eventNode *events.EventNode
)

type runner struct {
	Job          Job
	Config       JobConfig
	Trigger      Trigger
	TriggerIndex int
}

func init() {
	// TODO check if there is a config in couchdb first
	// parse configuration
	path := os.Getenv("JOB_CONFIG_LOCATION")
	if len(path) < 1 {
		path = "./config.json"
	}
	log.L.Infof("Parsing job configuration from %v", path)

	// read configuration
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.L.Fatalf("failed to read job configuration: %v", err)
	}

	// unmarshal job config
	var configs []JobConfig
	err = json.Unmarshal(b, &configs)
	if err != nil {
		log.L.Fatalf("unable to parse job configuration: %v", err)
	}

	// validate all jobs exist
	for _, config := range configs {
		if !config.Enabled {
			log.L.Debugf("Skipping %v, because it's disabled.", config.Name)
			continue
		}

		if _, ok := jobs[config.Name]; !ok {
			log.L.Fatalf("job %v doesn't exist.", config.Name)
		}

		// build a runner for each trigger
		for i, trigger := range config.Triggers {
			runner := &runner{
				Job:          jobs[config.Name],
				Config:       config,
				Trigger:      trigger,
				TriggerIndex: i,
			}

			// build regex if it's a match type
			if strings.EqualFold(runner.Trigger.Type, "match") {
				runner.buildMatchRegex()
			}

			log.L.Infof("Adding runner for job '%v', trigger #%v. Execution Type: %v", runner.Config.Name, runner.TriggerIndex, runner.Trigger.Type)
			runners = append(runners, runner)
		}

	}
}

// StartJobScheduler starts the jobs in the job map
func StartJobScheduler() {
	// start event router
	eventRouter := os.Getenv("EVENT_ROUTER_ADDRESS")
	if len(eventRouter) == 0 {
		log.L.Fatalf("Event router address is not set.")
	}
	filters := []string{events.TestEnd, events.TestExternal}
	eventNode = events.NewEventNode("Device Monitoring", eventRouter, filters)

	workers := 10
	queue := 100

	log.L.Infof("Starting job scheduler. Running %v jobs with %v workers with a max of %v events queued at once.", len(jobs), workers, queue)
	wg := sync.WaitGroup{}

	var matchRunners []*runner
	for _, runner := range runners {
		switch runner.Trigger.Type {
		case "daily":
			go runner.runDaily()
		case "interval":
			go runner.runInterval()
		case "match":
			matchRunners = append(matchRunners, runner)
		default:
			log.L.Warnf("unknown trigger type '%v' for job %v|%v", runner.Trigger.Type, runner.Config.Name, runner.TriggerIndex)
		}
	}

	eventChan := make(chan events.Event, 300)
	go readEvents(eventChan)

	// start event processors
	for i := 0; i < workers; i++ {
		log.L.Debugf("Starting event processor %v", i)
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case event := <-eventChan:
					for i := range matchRunners {
						if matchRunners[i].doesEventMatch(&event) {
							go matchRunners[i].run(&event)
						}
					}
				}
			}
		}()
	}

	wg.Wait()
}

func readEvents(outChan chan events.Event) {
	for {
		event, err := eventNode.Read()
		if err != nil {
			log.L.Warnf("unable to read event from eventNode: %v", err)
			continue
		}

		outChan <- event
	}
}

func (r *runner) run(context interface{}) {
	log.L.Debugf("[%s|%v] Running job...", r.Config.Name, r.TriggerIndex)

	eventChan := make(chan events.Event, 100)
	go func() {
		for event := range eventChan {
			log.L.Debugf("Publishing event: %+v", event)
			eventNode.PublishEvent(events.APISuccess, event)
		}
	}()

	r.Job.Run(context, eventChan)
	close(eventChan)

	log.L.Debugf("[%s|%v] Finished.", r.Config.Name, r.TriggerIndex)
}

func (r *runner) runDaily() {
	tmpDate := fmt.Sprintf("2006-01-02T%s", r.Trigger.At)
	runTime, err := time.Parse(time.RFC3339, tmpDate)
	runTime = runTime.UTC()
	if err != nil {
		log.L.Warnf("unable to parse time '%s' to execute job %s daily. error: %s", r.Trigger.At, r.Config.Name, err)
		return
	}

	log.L.Infof("[%s|%v] Running daily at %s", r.Config.Name, r.TriggerIndex, runTime.Format("15:04:05 MST"))

	// figure out how long until next run
	now := time.Now()
	until := time.Until(time.Date(now.Year(), now.Month(), now.Day(), runTime.Hour(), runTime.Minute(), runTime.Second(), 0, runTime.Location()))
	if until < 0 {
		until = 24*time.Hour + until
	}

	log.L.Debugf("[%s|%v] Time to next run: %v", r.Config.Name, r.TriggerIndex, until)
	timer := time.NewTimer(until)

	for {
		<-timer.C
		r.run(nil)

		timer.Reset(24 * time.Hour)
	}
}

func (r *runner) runInterval() {
	interval, err := time.ParseDuration(r.Trigger.Every)
	if err != nil {
		log.L.Warnf("unable to parse duration '%s' to execute job %s on an interval. error: %s", r.Trigger.Every, r.Config.Name, err)
		return
	}

	log.L.Infof("[%s|%v] Running every %v", r.Config.Name, r.TriggerIndex, interval)

	ticker := time.NewTicker(interval)
	for range ticker.C {
		r.run(nil)
	}
}