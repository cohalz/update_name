package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	lambda_sdk "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/cohalz/anaconda"
)

//Credential is credential for Twitter API
type Credential struct {
	AccessToken       string `json:"accessToken"`
	AccessTokenSecret string `json:"accessTokenSecret"`
	ConsumerKey       string `json:"consumerKey"`
	ConsumerSecret    string `json:"consumerSecret"`
}

//Event is Input for Lambda
type Event struct {
	Rules      []Rule     `json:"rules"`
	Credential Credential `json:"credential"`
}

//Rule is rules for update_name
type Rule struct {
	TriggerType     string `json:"triggerType"`
	TriggerWord     string `json:"triggerWord"`
	OmitTriggerWord bool   `json:"omitTriggerWord"`
	ReplyFormat     string `json:"replyFormat"`
}

func main() {

	lambda.Start(handleLambdaEvent)

}

func handleLambdaEvent(ctx context.Context, e Event) error {

	api := getAPIFromCredential(e.Credential)

	user, err := api.GetSelf(url.Values{})
	if err != nil {
		log.Fatal(err)
	}

	tweets := getTimeLine(api, user.ScreenName)

	checkTweetsAndUpdateName(api, tweets, e.Rules)

	functionName := lambdacontext.FunctionName
	if functionName != "test" {

		setSinceIDToEnv(functionName, user.ScreenName, tweets[0].Id)
	}

	return nil
}

func getAPIFromCredential(credential Credential) *anaconda.TwitterApi {

	return anaconda.NewTwitterApiWithCredentials(
		credential.AccessToken,
		credential.AccessTokenSecret,
		credential.ConsumerKey,
		credential.ConsumerSecret,
	)
}

func getTimeLine(api *anaconda.TwitterApi, sceenName string) []anaconda.Tweet {
	v := url.Values{}
	v.Set("count", "200")
	v.Set("include_rts", "false")
	sinceID, exists := os.LookupEnv("sinceID_" + sceenName)
	if exists {
		v.Set("since_id", sinceID)
	}

	tweets, err := api.GetHomeTimeline(v)
	if err != nil {
		log.Fatal(err)
	}

	return tweets
}

func checkTweetsAndUpdateName(api *anaconda.TwitterApi, tweets []anaconda.Tweet, rules []Rule) {
	for _, tweet := range tweets {
		for _, rule := range rules {
			if textIsMatchTrigger(tweet.FullText, rule) {
				updateTwitter(api, tweet, rule)
				break
			}
		}
	}
}

func textIsMatchTrigger(text string, rule Rule) bool {
	switch rule.TriggerType {
	case "prefix":
		return strings.HasPrefix(text, rule.TriggerWord)
	case "suffix":
		return strings.HasSuffix(text, rule.TriggerWord)
	case "ng":
		return strings.Contains(text, rule.TriggerWord)
	default:
		return false
	}
}

func updateTwitter(api *anaconda.TwitterApi, tweet anaconda.Tweet, rule Rule) {
	newName := tweet.FullText

	if newName[0:1] == "@" {
		newName = strings.SplitN(newName, " ", 2)[1]
	}

	if rule.OmitTriggerWord {
		newName = strings.Replace(newName, rule.TriggerWord, "", -1)
	}

	if rule.TriggerType == "ng" {
		newTweet := fmt.Sprintf("@" + tweet.User.ScreenName + " " + rule.ReplyFormat)
		v := url.Values{}
		v.Set("in_reply_to_status_id", strconv.FormatInt(tweet.Id, 10))
		api.PostTweet(newTweet, v)
		return
	}

	if utf8.RuneCountInString(newName) > 50 {
		return
	}

	api.Favorite(tweet.Id)

	updateProfile(api, newName)

	if rule.ReplyFormat != "" {
		newTweet := fmt.Sprintf("@"+tweet.User.ScreenName+" "+rule.ReplyFormat, newName)
		v := url.Values{}
		v.Set("in_reply_to_status_id", strconv.FormatInt(tweet.Id, 10))
		api.PostTweet(newTweet, v)
	}
}

func updateProfile(api *anaconda.TwitterApi, newName string) {
	v := url.Values{}
	v.Set("name", newName)

	_, err := api.PostAccountUpdateProfile(v)

	if err != nil {
		log.Fatal(err)
	}
}

func setSinceIDToEnv(functionName string, screenName string, sinceID int64) {
	sinceIDStr := strconv.FormatInt(sinceID, 10)

	sess := session.Must(session.NewSession())

	svc := lambda_sdk.New(
		sess,
		aws.NewConfig().WithRegion("ap-northeast-1"),
	)

	m := make(map[string]*string)

	envs := os.Environ()

	for _, env := range envs {
		if !strings.HasPrefix(env, "sinceID_") {
			continue
		}
		envKeyValue := strings.SplitN(env, "=", 2)
		m[envKeyValue[0]] = &envKeyValue[1]
	}

	m["sinceID_"+screenName] = &sinceIDStr

	env := &lambda_sdk.Environment{
		Variables: m,
	}

	input := &lambda_sdk.UpdateFunctionConfigurationInput{
		FunctionName: &functionName,
		Environment:  env,
	}

	_, err := svc.UpdateFunctionConfiguration(input)

	if err != nil {
		log.Fatal(err)
	}

}
