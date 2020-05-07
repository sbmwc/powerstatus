package function

import (
	"encoding/json"
	"fmt"
	"log"
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

type Credentials struct {
	JsonCredentials string
	JsonToken       string
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

	// get credentials and token from DB
	k := datastore.NameKey("credentials", "gmail-credentials", nil)
	var credentials Credentials
	if err := datastoreClient.Get(ctx, k, &credentials); err != nil {
		log.Fatalf("Failed to get DB credentials: %v\n", err)
	}

	config, err := google.ConfigFromJSON([]byte(credentials.JsonCredentials), common.GetNeededScopes()[:]...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to auth2.config: %v\n", err)
	}

	tok := &oauth2.Token{}
	json.Unmarshal([]byte(credentials.JsonToken), tok)

	httpClient = config.Client(ctx, tok)

	// looks like this would return the updated token if it is really needed (not clear)
	//	updatedToken, _ := config.TokenSource(ctx, tok).Token()
	//	fmt.Printf("token after:%v\n", *updatedToken)

	// read DB to get options
	k = datastore.NameKey("options", "function-options", nil)
	if err := datastoreClient.Get(ctx, k, &options); err != nil {
		log.Fatalf("Failed to get function options: %v\n", err)
	}

	fmt.Printf("Successfully inititailzed from DB\n")
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

// return true if either:
// 1. DB had NoError() and newErrorString is not NoError() or
// 2. DB had an error (not NoError()) and newErrorString is NoError()
// (IOWs, return true if DB flipped from NoError() to an error string or visa-versa)
func storeErrorString(newErrorString string) (bool, error) {
	var lastErrorInDB ErrorData
	k := datastore.NameKey("errors", "last-error", nil)
	err := datastoreClient.Get(ctx, k, &lastErrorInDB)
	if err != nil {
		return false, err
	}

	if newErrorString == NoError() {
		if lastErrorInDB.ErrorString == NoError() {
			return false, nil // no change
		} else {
			lastErrorInDB.ErrorString = NoError()
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return true, nil // changed
		}
	} else {
		if lastErrorInDB.ErrorString == NoError() {
			lastErrorInDB.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return true, nil // changed
		} else {
			lastErrorInDB.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return false, nil // error in DB was changed, but not to NoError()
		}
	}
}
