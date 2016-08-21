package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

var (
	token       = flag.String("token", "", "Token for Slack")
	fromChannel = flag.String("from", "", "Name of channel where to look for messages")
	toChannel   = flag.String("to", "", "Name of channel where to post messages")
	slackUser   = flag.String("user", "", "User name for Slack")
	debug       = flag.Bool("debug", false, "Enable debug mode")

	keywords             = []string{"ваканси", "работа", "позици", "тестировщик", "автоматизатор", "должность", "требования"}
	URLKeywords          = []string{"hh.ru", "job", "linkedin", "position", "vacancy", "work", "career"}
	fromID, toID, userID string
)

const regex = "(http|https)://([\\w_-]+(?:(?:\\.[\\w_-]+)+))([\\w.,@?^=%&:/~+#-]*[\\w@?^=%&/~+#-])?"

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetPrefix("slack-bot:")
	if *token == "" || *fromChannel == "" || *toChannel == "" || *slackUser == "" {
		fmt.Println("Specify correct flags")
		flag.PrintDefaults()
		os.Exit(1)
	}

	r, err := regexp.Compile(regex)
	if err != nil {
		log.Fatal("Can't compile regexp:", err)
	}

	api := slack.New(*token)
	api.SetDebug(*debug)

	users, err := api.GetUsers()
	if err != nil {
		log.Println("Can't get list of users:", err)
		return
	}
	for _, user := range users {
		if user.Name == *slackUser {
			userID = user.ID
			break
		}
	}

	channels, err := api.GetChannels(false)
	if err != nil {
		log.Println("Can't get list of channels:", err)
		return
	}
	for _, channel := range channels {
		switch channel.Name {
		case *fromChannel:
			fromID = channel.ID
		case *toChannel:
			toID = channel.ID
		default:
		}
	}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Println("Event Received")
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				if ev.Channel == fromID {
					text := ev.Text
					if ev.SubMessage != nil {
						text = ev.SubMessage.Text
					}
					if isJobPosting(text, r) {
						text = strings.Replace(text, "<", "", -1)
						text = strings.Replace(text, ">", "", -1)
						rtm.SendMessage(rtm.NewOutgoingMessage(text, toID))
					}
				}
				if ev.Channel == toID {
					if ev.User != userID {
						api.DeleteMessage(toID, ev.Timestamp)
					}
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Println("Error: Invalid credentials!")
				break

			default:
			}
		}
	}

}

func isJobPosting(text string, r *regexp.Regexp) bool {
	text = strings.ToLower(text)
	return r.MatchString(text) && (containsKeyword(text, keywords) || containsKeyword(text, URLKeywords))
}

func containsKeyword(text string, list []string) bool {
	result := false
	for _, v := range list {
		if strings.Contains(text, v) {
			result = true
			break
		}
	}
	return result
}
