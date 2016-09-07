package main

import (
	"math/rand"
	"regexp"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/nlopes/slack"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// TestClient test implementation of Slacker interface
type TestClient struct {
}

func (c TestClient) Repost(toID, text string) error {
	return nil
}

func (c TestClient) Delete(toID, timestamp string) error {
	return nil
}

func TestRegexp(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"[skype - tamara.mishcherina ]\nВсем привет! Открылась вакансия для QA automation (опыт от 3+) на удаленку. English level (speaking, writing, reading) - intermediate level. Автоматизация на С#. Пишите в личку, отвечу на все вопросы. :slightly_smiling_face:", false},
		{"htttp://hh.ru/dfffgfgf", false},
		{"http://hh.ru/dfffgfgf", true},
		{"something http://example.com/jobs", true},
		{"http://example.com/jobs dfdf f- dfd ", true},
		{"dsssdsdsd http://example.com  dfdf f- dfd ", true},
	}

	r, _ := regexp.Compile(REGEX)

	for _, v := range cases {
		result := r.MatchString(v.in)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestContainsKeyword(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"dfd/", false},
		{"htttp://hh.ru/dfffgfgf", true},
		{"http://example.com/jobs", true},
	}

	for _, v := range cases {
		result := containsKeyword(v.in, linkKeywords)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestIsJobPosting(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"[skype - tamara.mishcherina ]\nВсем привет! Открылась вакансия для QA automation (опыт от 3+) на удаленку. English level (speaking, writing, reading) - intermediate level. Автоматизация на С#. Пишите в личку, отвечу на все вопросы. :slightly_smiling_face:", false},
		{"http://hh.ru/something", true},
		{"something interesting http://example.com/jobs", true},
	}

	r, _ := regexp.Compile(REGEX)

	for _, v := range cases {
		result := isJobPosting(v.in, r)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestSlackClientRepost(t *testing.T) {
	cases := []struct {
		msg       *slack.MessageEvent
		res, desc string
	}{
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "222",
			},
		}, WRONG_CHANNEL_ID, "Wrong ID"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "111",
				Text:    "wqeqewqewq",
			},
		}, NOT_JOB_POSTING, "Incorrect text"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "111",
				Text:    "http://hh.ru",
			},
			SubMessage: &slack.Msg{
				Text: "wqeqweqeew",
			},
		}, NOT_JOB_POSTING, "Initially correct text, but incorrect after transformation"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel:     "111",
				Text:        "http://hh.ru",
				Attachments: []slack.Attachment{{}},
			},
		}, NOT_JOB_POSTING, "Correct text with attachments"},
	}

	fromID = "111"
	r, _ := regexp.Compile(REGEX)
	client := &SlackClient{
		Client: TestClient{},
	}

	for _, v := range cases {
		err := client.RepostMessage(v.msg, r)
		if err == nil {
			t.Errorf("For case: %s, error shouldn't be nil!", v.desc)
		}
		if err != nil && err.Error() != v.res {
			t.Errorf("For case: %s, actual error:%s, expected: %s", v.desc, err.Error(), v.res)
		}
	}
}

func TestAlreadyPostedMessageShouldntBePostedTwice(t *testing.T) {
	// Initialization of test DB
	db, err := bolt.Open("test.db", 0600, nil)
	if err != nil {
		t.Fatal("Can't open DB: ", err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BUCKET))
		return err
	})
	if err != nil {
		t.Fatal("Can't create bucket: ", err)
	}

	// Post double message
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BUCKET))
		err := bucket.Put([]byte("test http://hh.ru"), []byte("test http://hh.ru"))
		return err
	})
	if err != nil {
		t.Fatal("Can't add entry: ", err)
	}

	// Prepare test data
	fromID = "111"
	r, _ := regexp.Compile(REGEX)
	ev := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "111",
			Text:    "test http://hh.ru",
		},
	}
	client := &SlackClient{
		Client:  TestClient{},
		Storage: db,
	}

	err = client.RepostMessage(ev, r)
	if err == nil {
		t.Error("Should return an error on attempt to repost already reposted message")
	}
}

func TestNewMessageShouldBeReposted(t *testing.T) {
	// Initialization of test DB
	db, err := bolt.Open("test.db", 0600, nil)
	if err != nil {
		t.Fatal("Can't open DB: ", err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BUCKET))
		return err
	})
	if err != nil {
		t.Fatal("Can't create bucket: ", err)
	}

	// Prepare test data
	fromID = "111"
	r, _ := regexp.Compile(REGEX)
	ev := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "111",
			Text:    "test http://hh.ru " + randomString(50),
		},
	}
	client := &SlackClient{
		Client:  TestClient{},
		Storage: db,
	}

	err = client.RepostMessage(ev, r)
	if err != nil {
		t.Error("Should return ok for repost of a new message")
	}

}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
