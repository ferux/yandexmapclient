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

func main() {
	client, err := yandexmapclient.New(yandexmapclient.WithLogger(Logger{}))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Enter your stop id to get info")
	var stopID string
	for {
		_, err = fmt.Scanln(&stopID)
		if err != nil {
			log.Print(err)
		}

		if len(stopID) == 0 {
			log.Print("exiting")
			break
		}

		if !strings.HasPrefix(stopID, "stop__") {
			stopID = "stop__" + stopID
		}

		info, err := client.FetchStopInfo(stopID)
		if err != nil {
			log.Printf("error fetching info: %v", err)
			continue
		}

		for _, tr := range info.Data.Properties.StopMetaData.Transport {
			t, err := time.Parse("15:04", tr.BriefSchedule.DepartureTime)
			if err != nil {
				continue
			}
			// cur := time.Now().UTC()
			// t = t.AddDate(cur.Year(), int(cur.Month()-1), cur.Day())
			// dur := t.Sub(cur)
			arrival := t.Hour()*60 + t.Minute()
			log.Printf("%5s will arrive in %4d minutes", tr.Name, arrival)
		}

		_ = json.NewEncoder(os.Stdout).Encode(&info)
	}
}

type Logger struct{}

func (Logger) Debug(msg string) {
	log.Print(msg)
}

func (Logger) Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
