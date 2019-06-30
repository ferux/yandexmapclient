package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ferux/yandexmapclient"
)

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

		info, err := client.FetchStopInfo(command, prognosis)
		if err != nil {
			fmt.Printf("error fetching info: %v\n", err)
			continue
		}

		if len(info.Data.Properties.StopMetaData.Transport) == 0 {
			fmt.Println("no transport found")
			continue
		}

		for _, tr := range info.Data.Properties.StopMetaData.Transport {
			t, err := time.Parse("15:04", tr.BriefSchedule.DepartureTime)
			if err != nil {
				continue
			}
			arrival := t.Hour()*60 + t.Minute()
			fmt.Printf("%5s will arrive in %4d minutes\n", tr.Name, arrival)
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
	for k, _ := range handlers {
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
