package main

import (
	"regexp"
	"testing"
)

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

	r, _ := regexp.Compile(regex)

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
		{"something interesting http://example.slack.com/jobs", false},
	}

	r, _ := regexp.Compile(regex)

	for _, v := range cases {
		result := isJobPosting(v.in, r)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}

func TestIsNotSlackURL(t *testing.T) {
	cases := []struct {
		in  string
		res bool
	}{
		{"fgfvfvfvfvfv", true},
		{"https://softwaretesters.slack.com/", false},
	}

	for _, v := range cases {
		result := isNotSlackURL(v.in)
		if result != v.res {
			t.Errorf("For string: %s, actual result: %v, expected: %v", v.in, result, v.res)
		}
	}
}
