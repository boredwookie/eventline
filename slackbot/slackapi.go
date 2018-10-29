package slackbot

import (
	"fmt"
	"github.com/boredwookie/eventline/models"
	"github.com/volatiletech/null"
	"log"
	"os"

	"github.com/nlopes/slack"
)

var slackToken string
var userIdsToName map[string]string

func init() {
	// Ensure that the map is initialized _ONLY ONCE_
	userIdsToName = make(map[string]string)
}

//
// Reads the starred messages for the user. Can be polled no more frequently than every 2 seconds!
//	Requires a USER token (_NOT_ a 'bot token')!
func LoadStars(slackUserToken string) []models.Record{
	slackToken = slackUserToken

	api := slack.New(
		slackUserToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	stars, _, _ := api.ListStars(slack.StarsParameters{})

	var starredMessageRecords []models.Record
	for _, star := range stars{
		sourceRecordId := star.Channel + ":" + star.Message.Msg.User + ":" + star.Message.Msg.Timestamp

		// Get the user's name (if it isn't already in our name cache)
		if _, ok := userIdsToName[star.Message.Msg.User]; !ok{
			if getUserInfo(star.Message.Msg.User) == nil{
				fmt.Println("Missing required scope `users.profile:read`")
				os.Exit(32)
			}
			userIdsToName[star.Message.Msg.User] = getUserInfo(star.Message.Msg.User).RealName
		}

		starredMessageRecords = append(starredMessageRecords, models.Record{
			SourceRecordId: sourceRecordId,
			Identity: null.StringFrom(userIdsToName[star.Message.Msg.User]),
			SourceType: null.StringFrom("slack"),
			Notes:null.StringFrom(star.Message.Msg.Text),
		})
	}

	return starredMessageRecords
}

//
// Takes a slack user identity and returns the real name
func getUserInfo(slackUserId string) *slack.User{
	api := slack.New(
		slackToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	userInfo, _ := api.GetUserInfo(slackUserId)
	return userInfo
}