package sumorus

import (
	"github.com/sirupsen/logrus"
	"time"
	"strings"
	"encoding/json"
	"net/http"
	"bytes"

	"fmt"
)

type SumoLogicHook struct {
	endPointUrl	   	string
	tags			[]string
	host			string
	levels 			[]logrus.Level

	messageChan chan []byte
}

type SumoLogicMesssage struct {
	Tags			[]string		`json:"tags"`
	Host			string			`json:"host"`
	Level			string			`json:"level"`
	Data			interface{}		`json:"data"`
}

var (
	newline = []byte{'\n'}
)

func NewSumoLogicHook(endPointUrl string, host string, level logrus.Level, tags ...string) *SumoLogicHook {
	levels := []logrus.Level{}
	for _, l := range []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	} {
		if l <= level {
			levels = append(levels, l)
		}
	}

	var tagList []string
	for _, tag := range tags {
		tagList = append(tagList,tag)
	}

	hook := &SumoLogicHook{
		host: host,
		tags: tagList,
		endPointUrl: endPointUrl,
		levels: levels,
		messageChan: make(chan []byte, 1000),
	}
	go hook.run()
	return hook
}

func (hook *SumoLogicHook) Fire(entry *logrus.Entry) error {
	data := map[string]interface{}{
		"message": entry.Message,
		"fields": entry.Data,
	}

	message := SumoLogicMesssage{
		Tags: hook.tags,
		Host: hook.host,
		Level: strings.ToUpper(entry.Level.String()),
		Data: data,
	}
	payload, _ := json.Marshal(message)
	hook.messageChan <- payload

	return nil
}

func (hook *SumoLogicHook) send(messages [][]byte) {
	req, err := http.NewRequest(
		"POST",
		hook.endPointUrl,
		bytes.NewBuffer(bytes.Join(messages, newline)),
	)
	if err != nil {
		fmt.Println("error creating sumologic request", err)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("error sending sumologic request", err)
		return
	}

	resp.Body.Close()
}

func (hook *SumoLogicHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *SumoLogicHook) run() {
	// Batch send the current log lines each 15 seconds
	tock := time.NewTicker(15 * time.Second)
	var messages [][]byte
	for {
		select {
		case <-tock.C:
			if len(messages) == 0 {
				continue
			}
			go hook.send(messages)
			messages = nil

		case message := <-hook.messageChan:
			messages = append(messages, message)
		}
	}
}
