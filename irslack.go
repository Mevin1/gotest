package main

/**
 * linkr bridges Slack and IRC
 */

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"dependencies/slack"
	"dependencies/go-ircevent"
)

var (
	SlackAPIToken = "xoxb-6658913443-UDe4y08tcI83E5gz83IEtwBi"
	SlackChannel  = "#general"
	IRCNick       = "Mevin1"
	IRCPassword   = "kivin"
	IRCNetwork    = "irc.rizon.net:6660"
	IRCChannel    = "#test"
	DebugMode     = false
)

var (
	SlackAPI *slack.Slack
	IRCLink  *irc.Connection
)

func unescapeMessage(message string) string {
	replacer := strings.NewReplacer("&amp", "&", "&lt;", "<", "&gt;", ">")
	return replacer.Replace(message)
}

func PipeSlackToIRC(slackAPI *slack.Slack, ircLink *irc.Connection) {
	//sender := make(chan slack.OutgoingMessage)
	receiver := make(chan slack.SlackEvent)
	wsAPI, err := slackAPI.StartRTM("", "http://example.com")
	if err != nil {
		log.Fatalf("StartRTM() error: %s", err)
	}
	go wsAPI.HandleIncomingEvents(receiver)
	go wsAPI.Keepalive(10 * time.Second)
	for {
		msg := <-receiver
		switch msg.Data.(type) {
		case *slack.MessageEvent:
			msgEvent := msg.Data.(*slack.MessageEvent)
			// Ignore bot messages, including our own
			if msgEvent.BotId != "" {
				break
			}

			fmt.Printf("Message: %s\n", msgEvent)
			user, err := slackAPI.GetUserInfo(msgEvent.UserId)
			if err != nil {
				log.Printf("GetUserInfo(): %s\n", err)
				break
			}
			msg := fmt.Sprintf("(Slack) <%s> %s", user.Profile.RealName, unescapeMessage(msgEvent.Text))
			ircLink.Privmsg(IRCChannel, msg)
			fmt.Println("Slack -> IRC:", msg)
		}
	}
}

func SendIRCToSlack(event *irc.Event, slackAPI *slack.Slack) {
	params := slack.PostMessageParameters{
		Username: fmt.Sprintf("(IRC) %s", event.Nick),
		AsUser:   true,
	}
	_, _, err := slackAPI.PostMessage(SlackChannel, event.Message(), params)
	if err != nil {
		log.Println("SendIRCToSlack:", err)
	} else {
		fmt.Println("IRC -> Slack:", event.Message())
	}
}

func main() {
	// Connect to Slack
	SlackAPI = slack.New(SlackAPIToken)
	SlackAPI.SetDebug(DebugMode)

	// Connect to IRC
	IRCLink = irc.IRC(IRCNick, IRCNick)
	IRCLink.UseTLS = true
	IRCLink.Password = IRCPassword
	IRCLink.Connect(IRCNetwork)
	IRCLink.Join(IRCChannel)

	// Setup Callbacks
	go PipeSlackToIRC(SlackAPI, IRCLink)
	IRCLink.AddCallback("PRIVMSG",
		func(event *irc.Event) {
			SendIRCToSlack(event, SlackAPI)
		})

	// Loop FOREVER.
	for {
		<-time.After(5 * time.Second)
	}
}
