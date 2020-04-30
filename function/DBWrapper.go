package function

import (
	"encoding/json"
	"fmt"
	//       "html"
	"net/http"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"sbmwc/powerstatus/common"
)

var ctx context.Context
var datastoreClient *datastore.Client
var httpClient *http.Client
var options FunctionOptions

type ClientCredentials struct {
	JsonCredentials string
}

type Token struct {
	JsonToken string
}

type FunctionOptions struct {
	LabelId string
	DocId   string
}

type ErrorData struct {
	ErrorString string
}

func init() {
	ctx = context.Background()
	datastoreClient, _ = datastore.NewClient(ctx, datastore.DetectProjectID)

	k := datastore.NameKey("credentials", "user-credentials", nil)
	var clientCredentials ClientCredentials
	if err := datastoreClient.Get(ctx, k, &clientCredentials); err != nil {
		fmt.Printf("Failed to get value: %v", err)
	}
	fmt.Printf("JsonCredentials from DB:%v\n", clientCredentials)

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON([]byte(clientCredentials.JsonCredentials), common.GetNeededScopes()[:]...)
	if err != nil {
		fmt.Printf("Unable to parse client secret file to config: %v", err)
	}

	//read token from DB
	k = datastore.NameKey("tokens", "user-token", nil)
	var token Token
	if err := datastoreClient.Get(ctx, k, &token); err != nil {
		fmt.Printf("Failed to get token: %v", err)
	}
	fmt.Printf("json token from DB:%s\n", token.JsonToken)
	tok := &oauth2.Token{}
	json.Unmarshal([]byte(token.JsonToken), tok)
	fmt.Printf("token before:%v\n", *tok)

	httpClient = config.Client(ctx, tok)

	// looks like this would return the updated token if it is really needed (not clear)
	//	updatedToken, _ := config.TokenSource(ctx, tok).Token()
	//	fmt.Printf("token after:%v\n", *updatedToken)

	// read DB to get options
	k = datastore.NameKey("options", "function-options", nil)
	if err := datastoreClient.Get(ctx, k, &options); err != nil {
		fmt.Printf("Failed to get value: %v", err)
	}
}

func getHTTPClientUsingDB() *http.Client {
	return httpClient
}

func getOptionsFromDB() *FunctionOptions {
	return &options
}

var noError string = "<none>"

func NoError() string {
	return noError
}

// return true if what is currently stored is NoError()
func storeErrorString(newErrorString string) (bool, error) {
	var errorData ErrorData
	k := datastore.NameKey("errors", "last-error", nil)
	err := datastoreClient.Get(ctx, k, &errorData)
	if err != nil {
		return false, err
	}

	if newErrorString == NoError() {
		if errorData.ErrorString == NoError() {
			return false, nil // no change
		} else {
			errorData.ErrorString = NoError()
			datastoreClient.Put(ctx, k, &errorData)
			return true, nil // changed
		}
	} else {
		if errorData.ErrorString == NoError() {
			errorData.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &errorData)
			return true, nil // changed
		} else {
			errorData.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &errorData)
			return false, nil // error in DB was updated (technically changed, but not to NoError()
		}
	}
}
