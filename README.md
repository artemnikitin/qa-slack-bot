# qa-slack-bot    
[![Go Report Card](https://goreportcard.com/badge/github.com/artemnikitin/qa-slack-bot)](https://goreportcard.com/report/github.com/artemnikitin/qa-slack-bot)    [![codebeat badge](https://codebeat.co/badges/4c60f84b-2810-4047-929d-c8609569533a)](https://codebeat.co/projects/github-com-artemnikitin-qa-slack-bot)    [![Build Status](https://travis-ci.org/artemnikitin/qa-slack-bot.svg?branch=master)](https://travis-ci.org/artemnikitin/qa-slack-bot)    
Bot for softwaretesters.slack.com

#### How to run it
``` 
qa-slack-bot -token slack_token -from channel_name -to channel_name -user user_name(bot_name)
```
Parameters:
- `token`    
Slack token
- `from`    
Name of channel where bot will get info
- `to`    
Name of channel where bot will repost message
- `user`    
User (bot) name that will be displayed in Slack
