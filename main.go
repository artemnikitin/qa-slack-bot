package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/nlopes/slack"
)

var (
	token       = flag.String("token", "", "Token for Slack")
	fromChannel = flag.String("from", "", "Name of channel where to look for messages")
	toChannel   = flag.String("to", "", "Name of channel where to post messages")
	slackUser   = flag.String("user", "", "User name for Slack")
	debug       = flag.Bool("debug", false, "Enable debug mode")

	textKeywords         = []string{"ваканси", "работа", "позици", "тестировщик", "автоматизатор", "должность", "требования"}
	linkKeywords         = []string{"hh.ru", "job", "linkedin.com/jobs", "position", "vacancy", "work", "career"}
	exclusions           = []string{".slack.com", "linkedin.com/comm/profile"}
	fromID, toID, userID string
	userMap              map[string]string
)

const (
	bucket   = "QA-SLACK"
	regexURL = "(http|https)://([\\w_-]+(?:(?:\\.[\\w_-]+)+))([\\w.,@?^=%&:/~+#-]*[\\w@?^=%&/~+#-])?"
	//regexEmail             = "[A-Z0-9a-z._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,6}"
	wrongChannelID         = "Wrong channel ID"
	wrongUserID            = "Wrong user ID"
	messageIsNotJobPosting = "Not job posting"
	messageIsAlreadyPosted = "Already posted"
)

type slacker interface {
	Repost(string, string) error
	Delete(string, string) error
}

type slackerClient struct {
	Slack *slack.Client
}

func (c slackerClient) Repost(toID, text string) error {
	params := slack.PostMessageParameters{
		AsUser: true,
	}
	_, _, err := c.Slack.PostMessage(toID, text, params)
	return err
}

func (c slackerClient) Delete(toID, timestamp string) error {
	_, _, err := c.Slack.DeleteMessage(toID, timestamp)
	return err
}

type slackClient struct {
	Client  slacker
	Storage *bolt.DB
}

func (c *slackClient) RepostMessage(ev *slack.MessageEvent, r *regexp.Regexp) error {
	if ev.Channel != fromID {
		return errors.New(wrongChannelID)
	}
	if len(ev.Attachments) > 0 {
		return errors.New(messageIsNotJobPosting)
	}
	text := ev.Text
	if ev.SubMessage != nil && ev.SubMessage.Text != "" {
		text = ev.SubMessage.Text
	}
	if !isJobPosting(text, r) {
		return errors.New(messageIsNotJobPosting)
	}
	text = strings.Replace(text, "<", "", -1)
	text = strings.Replace(text, ">", "", -1)
	if strings.Contains(text, "@U") {
		text = replaceIDWithNickname(text)
	}
	if alreadyPosted(text, c.Storage) {
		return errors.New(messageIsAlreadyPosted)
	}
	savePosted(text, c.Storage)
	err := c.Client.Repost(toID, text)
	return err
}

func (c *slackClient) DeleteMessage(ev *slack.MessageEvent) error {
	if ev.Channel != toID {
		return errors.New(wrongChannelID)
	}
	if ev.User == userID {
		return errors.New(wrongUserID)
	}
	err := c.Client.Delete(toID, ev.Timestamp)
	return err
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	userMap = make(map[string]string)

	if *token == "" || *fromChannel == "" || *toChannel == "" || *slackUser == "" {
		fmt.Println("Specify correct flags")
		flag.PrintDefaults()
		os.Exit(1)
	}

	db, err := bolt.Open("repost.db", 0600, nil)
	if err != nil {
		log.Fatal("Can't open DB: ", err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		log.Fatal("Can't create bucket: ", err)
	}

	r, err := regexp.Compile(regexURL)
	if err != nil {
		log.Fatal("Can't compile regexp: ", err)
	}

	api := slack.New(*token)
	api.SetDebug(*debug)
	client := &slackClient{
		Client: slackerClient{
			Slack: api,
		},
		Storage: db,
	}

	getSlackUserID(api)
	getSlackChannelID(api)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		msg := <-rtm.IncomingEvents
		fmt.Println("Event Received")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			client.RepostMessage(ev, r)
			client.DeleteMessage(ev)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Println("Error: Invalid credentials!")
			break
		}
	}

}

func isJobPosting(text string, r *regexp.Regexp) bool {
	text = strings.ToLower(text)
	withKeyword := containsKeyword(text, textKeywords) || containsKeyword(text, linkKeywords)
	return r.MatchString(text) && withKeyword && validateExclusions(text)
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

func validateExclusions(text string) bool {
	for _, v := range exclusions {
		if strings.Contains(text, v) {
			return false
		}
	}
	return true
}

func replaceIDWithNickname(text string) string {
	if len(text) <= 10 {
		return text
	}
	count := strings.Count(text, "@U")
	place := 0
	start := 0
	for i := 0; i < count; i++ {
		if len(text[place:])-start >= 10 {
			start := strings.Index(text, "@U")
			place += start + 1
			id := text[start+1 : start+10]
			nickname, ok := userMap[id]
			if ok {
				text = strings.Replace(text, id, nickname, -1)
			}
		}
	}
	return text
}

func getSlackUserID(api *slack.Client) {
	users, err := api.GetUsers()
	if err != nil {
		log.Fatal("Can't get list of users:", err)
	}
	for _, user := range users {
		userMap[user.ID] = user.Name
		if user.Name == *slackUser {
			userID = user.ID
		}
	}
}

func getSlackChannelID(api *slack.Client) {
	channels, err := api.GetChannels(false)
	if err != nil {
		log.Fatal("Can't get list of channels:", err)
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
}

func alreadyPosted(text string, db *bolt.DB) bool {
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		bytes := bucket.Get([]byte(text))
		if bytes != nil {
			return errors.New("")
		}
		return nil
	})
	return err != nil
}

func savePosted(text string, db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		err := bucket.Put([]byte(text), []byte(text))
		return err
	})
	if err != nil {
		log.Println(err)
	}
}
