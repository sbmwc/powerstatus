package function

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	//"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"sbmwc/powerstatus/common"
)

const noError string = "<none>"

var ctx context.Context
var datastoreClient *datastore.Client
var httpClient *http.Client
var processor *common.EmailProcessor

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

type Credentials struct {
	JsonCredentials string
	JsonToken       string
}

type FunctionConfig struct {
	LabelId                    string
	DocId                      string
	ErrorAndWarnEmailAddresses string
}

// stores last error, or noError
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

	// get and cache processor
	processor, err = common.NewEmailProcessor(httpClient)
	if err != nil {
		log.Fatalf("ERROR:Could not create processor:%v\n", err)
	}

	fmt.Printf("Successfully inititailzed from DB\n")
}

// This is the method invoked by pub/sub
func PubSubProcessor(ctx context.Context, m PubSubMessage) error {
	//	name := string(m.Data)
	//	fmt.Printf("name:%s\n", name)   // name may be "" if nothing set in scheduler

	// read DB to get config
	// read every execution (vs init()) so that we can dynamcially change
	// configuration in DB and function will pick it up (vs cached)
	var functionConfig FunctionConfig
	k := datastore.NameKey("config", "function-config", nil)
	if err := datastoreClient.Get(ctx, k, &functionConfig); err != nil {
		log.Fatalf("Failed to get function config: %v\n", err)
	}

	executionStatus := processor.LookForAndProcessEmails(functionConfig.LabelId, functionConfig.DocId)

	if executionStatus.ErrString == "" {
		// success!
		fmt.Printf("Successfully processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)

		if b, _ := storeErrorString(noError); b {
			// previous run had an error, so send out notification of all-good
			okstr := "Successful run after previous error(s)"
			processor.AppendToGoogleDocs(functionConfig.DocId, okstr)
			processor.SendEmail(functionConfig.ErrorAndWarnEmailAddresses, "Powerstatus Script OK", okstr)
		}

		if len(executionStatus.WarnMsgs) > 0 {
			warnStr := fmt.Sprintf("Warnings:%v\n", executionStatus.WarnMsgs)
			processor.AppendToGoogleDocs(functionConfig.DocId, warnStr)
			processor.SendEmail(functionConfig.ErrorAndWarnEmailAddresses, "Powerstatus Script Warning", warnStr)
		}
	} else {
		// error!
		fmt.Printf("ERROR:%s\n", executionStatus.ErrString)

		if b, _ := storeErrorString(executionStatus.ErrString); b {
			// first error -- i.e., no previous error so send out notification
			processor.AppendToGoogleDocs(functionConfig.DocId, "ERROR:"+executionStatus.ErrString)
			processor.SendEmail(functionConfig.ErrorAndWarnEmailAddresses, "Powerstatus Script Error!", "ERROR:"+executionStatus.ErrString)
		}
	}
	return nil
}

// return true if either:
// 1. DB had noError and newErrorString is not noError or
// 2. DB had an error (not noError) and newErrorString is noError
// (IOWs, return true if DB flipped from noError to an error string or visa-versa)
func storeErrorString(newErrorString string) (bool, error) {
	var lastErrorInDB ErrorData
	k := datastore.NameKey("errors", "last-error", nil)
	err := datastoreClient.Get(ctx, k, &lastErrorInDB)
	if err != nil {
		return false, err
	}

	if newErrorString == noError {
		if lastErrorInDB.ErrorString == noError {
			return false, nil // no change
		} else {
			lastErrorInDB.ErrorString = noError
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return true, nil // changed
		}
	} else {
		if lastErrorInDB.ErrorString == noError {
			lastErrorInDB.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return true, nil // changed
		} else {
			lastErrorInDB.ErrorString = newErrorString
			datastoreClient.Put(ctx, k, &lastErrorInDB)
			return false, nil // error in DB was changed, but not to noError
		}
	}
}
