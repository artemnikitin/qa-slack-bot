package main

import (
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/nlopes/slack"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type testClient struct {
}

func (c testClient) Repost(toID, text string) error {
	return nil
}

func (c testClient) Delete(toID, timestamp string) error {
	return nil
}

func TestRegexp(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"htttp://hh.ru/dfffgfgf", false},
		{"http://hh.ru/dfffgfgf", true},
		{"something http://example.com/jobs", true},
		{"http://example.com/jobs dfdf f- dfd ", true},
		{"dsssdsdsd http://example.com  dfdf f- dfd ", true},
	}

	r, _ := regexp.Compile(regexURL)

	for _, v := range cases {
		result := r.MatchString(v.in)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestRegexpEmail(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"dfdfdfdfd", false},
		{"vasya@googl.ecom", true},
		{"vasya@googl.ecom something interesting", true},
		{"something vasya@goog.ckk", true},
		{"something vasya@goog.ckk something more", true},
	}

	r, _ := regexp.Compile(regexEmail)

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
		{"Нужен опытный секьюрити тестировщик для проверки веб приложения на Owasp Top 10 уязвимости. Если кому интересно, стучите в личку или присылайте свои CV на kmachuhin@gmail.com", true},
		{"http://hh.ru/something", true},
		{"something interesting http://example.com/jobs", true},
		{"[skype - tamara.mishcherina ]\nВсем привет! Открылась вакансия для QA automation (опыт от 3+) на удаленку. English level (speaking, writing, reading) - intermediate level. Автоматизация на С#. Пишите в личку, отвечу на все вопросы. :slightly_smiling_face:", true},
		{"Всем привет! Открылась вакансия для QA automation (опыт от 3+) на удаленку. English level (speaking, writing, reading) - intermediate level. Автоматизация на С#. Пишите в личку, отвечу на все вопросы. :slightly_smiling_face:", false},
		{"nice comment http://something.slack.com", false},
		{"something interesting http://example.com/jobs www.linkedin.com/comm/profile/fvfvf", false},
		{"job job job linkedin.com/profile/fvfvf", false},
	}

	var regexList []*regexp.Regexp
	r, _ := regexp.Compile(regexURL)
	regexList = append(regexList, r)
	r, _ = regexp.Compile(regexEmail)
	regexList = append(regexList, r)

	for _, v := range cases {
		result := isJobPosting(v.in, regexList)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestReplaceID(t *testing.T) {
	userMap = make(map[string]string)
	userMap["U22KZA25S"] = "vasya"
	userMap["U11KZA007"] = "aid"

	cases := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"@U22KZA25S test", "@vasya test"},
		{"@UQ1 test", "@UQ1 test"},
		{"@UQ1001200 test", "@UQ1001200 test"},
		{"test @U22KZA25S", "test @vasya"},
		{"test @U22KZA25S test", "test @vasya test"},
		{"@U22KZA25S test @U11KZA007", "@vasya test @aid"},
		{"@U22KZA25S @U11KZA007", "@vasya @aid"},
		{"@U22KZA25S@U11KZA007", "@vasya@aid"},
		{"test @U22KZA25S test @U11KZA007", "test @vasya test @aid"},
		{"test @U22KZA25S test @U11KZA007 test", "test @vasya test @aid test"},
		{"@U22KZA25S test @U11KZA007 test", "@vasya test @aid test"},
		{"@U22KZA25S test @U22KZA001", "@vasya test @U22KZA001"},
	}

	for _, v := range cases {
		result := replaceIDWithNickname(v.in)
		if result != v.out {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.out)
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
		}, wrongChannelID, "Wrong ID"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "111",
				Text:    "wqeqewqewq",
			},
		}, messageIsNotJobPosting, "Incorrect text"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "111",
				Text:    "http://hh.ru",
			},
			SubMessage: &slack.Msg{
				Text: "wqeqweqeew",
			},
		}, messageIsNotJobPosting, "Initially correct text, but incorrect after transformation"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel:     "111",
				Text:        "http://hh.ru",
				Attachments: []slack.Attachment{{}},
			},
		}, messageIsNotJobPosting, "Correct text with attachments"},
	}

	fromID = "111"
	var regexList []*regexp.Regexp
	r, _ := regexp.Compile(regexURL)
	regexList = append(regexList, r)
	r, _ = regexp.Compile(regexEmail)
	regexList = append(regexList, r)

	client := &slackClient{
		Client: testClient{},
	}

	for _, v := range cases {
		err := client.RepostMessage(v.msg, regexList)
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
		_, err = tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		t.Fatal("Can't create bucket: ", err)
	}

	// Post double message
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		err = bucket.Put([]byte("test http://hh.ru"), []byte("test http://hh.ru"))
		return err
	})
	if err != nil {
		t.Fatal("Can't add entry: ", err)
	}

	// Prepare test data
	fromID = "111"
	var regexList []*regexp.Regexp
	r, _ := regexp.Compile(regexURL)
	regexList = append(regexList, r)
	r, _ = regexp.Compile(regexEmail)
	regexList = append(regexList, r)

	ev := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "111",
			Text:    "test http://hh.ru",
		},
	}
	client := &slackClient{
		Client:  testClient{},
		Storage: db,
	}

	err = client.RepostMessage(ev, regexList)
	if err == nil {
		t.Error(err.Error())
	}
}

func TestNewMessageShouldBeReposted(t *testing.T) {
	userMap = make(map[string]string)
	userMap["U11KZA007"] = "aid"

	// Initialization of test DB
	db, err := bolt.Open("test.db", 0600, nil)
	if err != nil {
		t.Fatal("Can't open DB: ", err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		t.Fatal("Can't create bucket: ", err)
	}

	// Prepare test data
	fromID = "111"
	var regexList []*regexp.Regexp
	r, _ := regexp.Compile(regexURL)
	regexList = append(regexList, r)
	r, _ = regexp.Compile(regexEmail)
	regexList = append(regexList, r)

	ev := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "111",
			Text:    "test @U11KZA007 http://hh.ru " + randomString(50),
		},
	}
	client := &slackClient{
		Client:  testClient{},
		Storage: db,
	}

	err = client.RepostMessage(ev, regexList)
	if err != nil {
		t.Error(err.Error())
	}

}

func TestDeleteMessage(t *testing.T) {
	toID = "111"
	userID = "vasya"
	ev := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "111",
			User:    "kolya",
			Text:    "flood",
		},
	}
	client := &slackClient{
		Client: testClient{},
	}

	err := client.DeleteMessage(ev)
	if err != nil {
		t.Error("Can't delete message: ", err)
	}

}

func TestDeleteMessageIncorrectValues(t *testing.T) {
	cases := []struct {
		msg       *slack.MessageEvent
		res, desc string
	}{
		{&slack.MessageEvent{
			Msg: slack.Msg{
				User: "vasya",
				Text: "some flood",
			},
		}, wrongChannelID, "Wrong channel ID"},
		{&slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "111",
				User:    "vasya",
				Text:    "some flood",
			},
		}, wrongUserID, "Wrong user ID"},
	}

	toID = "111"
	userID = "vasya"
	client := &slackClient{
		Client: testClient{},
	}

	for _, v := range cases {
		err := client.DeleteMessage(v.msg)
		if err != nil && err.Error() != v.res {
			t.Errorf("For case: %s, actual error:%s, expected: %s", v.desc, err.Error(), v.res)
		}
	}
}

func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
