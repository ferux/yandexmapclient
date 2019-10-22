package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ferux/yandexmapclient"
)

const defaultTimeout = time.Second * 15

var handlers = map[string]func(){}

func main() {
	client, err := yandexmapclient.New(yandexmapclient.WithLogger(Logger{}))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Prognosis: false")

	var command string
	var prognosis bool

	handlers["p"] = func() {
		prognosis = !prognosis
		fmt.Println("Prognosis: ", prognosis)
	}
	handlers["exit"] = func() { os.Exit(0) }

	handlers["help"] = help
	handlers["prompt"] = prompt
	prompt()

	for {
		fmt.Print("command: ")
		_, err = fmt.Scanln(&command)
		if err != nil {
			log.Print(err)
		}

		if len(command) == 0 {
			fmt.Print("exiting")
			break
		}

		if !strings.HasPrefix(command, "stop__") {
			serveCommand(command)
			continue
		}

		var info yandexmapclient.StopInfo
		var err error

		func() {
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()
			info, err = client.FetchStopInfo(ctx, command, prognosis)
		}()

		if err != nil {
			fmt.Printf("error fetching info: %v\n", err)
			continue
		}

		if info.Data == nil {
			fmt.Println("data is nil")
			continue
		}

		if info.Data != nil {
			fmt.Println("current time is: ", info.Data.Properties.CurrentTime)
		}

		if len(info.Data.Properties.StopMetaData.Transport) == 0 {
			fmt.Println("no transport found")
			continue
		}

		for _, tr := range info.Data.Properties.StopMetaData.Transport {
			var pickTime time.Time
			bs := tr.Threads[0].BriefSchedule
			if len(bs.Events) > 0 {
				fmt.Println("picking scheduled time")
				pickTime = bs.Events[0].Scheduled.Time
				if pickTime.IsZero() {
					fmt.Println("picking estimated time")
					pickTime = bs.Events[0].Estimated.Time
				}
			} else {
				if time.Now().After(bs.Frequency.End.Time) {
					fmt.Println("picking begin time")
					pickTime = bs.Frequency.Begin.Time
				} else {
					pickTime = time.Now().Add(time.Second * time.Duration(bs.Frequency.Value))
				}
			}

			if pickTime.Before(time.Now()) {
				pickTime = pickTime.Add(time.Hour * 24)
			}

			fmt.Printf("%5s will arrive in %4.0f minutes at %s\n", tr.Name, time.Until(pickTime).Minutes(), pickTime)
		}

		_ = json.NewEncoder(os.Stdout).Encode(&info)
		fmt.Println()
	}
}

func serveCommand(command string) {

	f, ok := handlers[command]
	if !ok {
		fmt.Printf("command %q not found\n", command)
		return
	}

	f()
}

func prompt() {
	fmt.Println(strings.Repeat("*", 40))
	fmt.Println("Type help for available commands")
	fmt.Println("send empty line or type 'exit' to exit")
	fmt.Println(strings.Repeat("*", 40))
	fmt.Println("keep in mind that stop id should start with 'stop__' prefix")
}

func help() {
	fmt.Println(strings.Repeat("*", 40))
	fmt.Println("commands are:")
	for k := range handlers {
		fmt.Printf("%s ", k)
	}
	fmt.Println()
	// fmt.Println("stop__xxx -- where xxx is ID of the stop -- shows bus stop info")
	// fmt.Println("p -- switches prognosis mode for fetching bus stop info")
	// fmt.Println("help -- shows help")
}

type Logger struct{}

func (Logger) Debug(msg string) {
	log.Print(msg)
}

func (Logger) Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
