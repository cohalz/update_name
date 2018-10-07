package main

import (
	"context"
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
	AccessToken       string `json:"access_token"`
	AccessTokenSecret string `json:"access_token_secret"`
	ConsumerKey       string `json:"consumer_key"`
	ConsumerSecret    string `json:"consumer_secret"`
}

//Event is Input for Lambda
type Event struct {
	Trigger    Trigger    `json:"trigger"`
	Credential Credential `json:"credential"`
}

//Trigger is rules for update_name
type Trigger struct {
	PrefixRules []string `json:"prefixRules"`
	SuffixRules []string `json:"suffixRules"`
}

func main() {

	lambda.Start(handleLambdaEvent)

}

func handleLambdaEvent(ctx context.Context, e Event) error {
	api := getAPIFromCredential(e.Credential)

	tweets := getTimeLine(api)

	checkTweetsAndUpdateName(api, tweets, e.Trigger)

	functionName := lambdacontext.FunctionName
	if functionName != "test" {
		setSinceIDToEnv(functionName, tweets[0].Id)
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

func getTimeLine(api *anaconda.TwitterApi) []anaconda.Tweet {
	v := url.Values{}
	v.Set("count", "200")
	v.Set("include_rts", "false")
	sinceID, exists := os.LookupEnv("sinceID")
	if exists {
		v.Set("since_id", sinceID)
	}

	tweets, err := api.GetHomeTimeline(v)
	if err != nil {
		log.Fatal(err)
	}

	return tweets
}

func checkTweetsAndUpdateName(api *anaconda.TwitterApi, tweets []anaconda.Tweet, trigger Trigger) {

	for _, tweet := range tweets {
		text := tweet.FullText
		if utf8.RuneCountInString(text) > 50 {
			continue
		}

		for _, prefixRule := range trigger.PrefixRules {
			if strings.HasSuffix(text, prefixRule) {
				updateProfile(api, text)
			}
		}

		for _, suffixRule := range trigger.SuffixRules {
			if strings.HasSuffix(text, suffixRule) {
				updateProfile(api, text)
			}
		}

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

func setSinceIDToEnv(functionName string, sinceID int64) {
	sinceIDStr := strconv.FormatInt(sinceID, 10)

	sess := session.Must(session.NewSession())

	svc := lambda_sdk.New(
		sess,
		aws.NewConfig().WithRegion("ap-northeast-1"),
	)

	m := make(map[string]*string)
	m["sinceID"] = &sinceIDStr

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
