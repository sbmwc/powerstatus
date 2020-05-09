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
	NotificationEmailAddresses string
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

	fmt.Printf("Successfully inititailzed from DB\n")
}

// This is the method invoked by pub/sub
func PubSubProcessor(ctx context.Context, m PubSubMessage) error {
	operation := string(m.Data)

	// read DB to get config
	// read every execution (vs init()) so that we can dynamcially change
	// configuration in DB and function will pick it up (vs cached)
	var functionConfig FunctionConfig
	k := datastore.NameKey("config", "function-config", nil)
	if err := datastoreClient.Get(ctx, k, &functionConfig); err != nil {
		log.Fatalf("Failed to get function config: %v\n", err)
	}

	processor, err := common.NewEmailProcessor(httpClient, functionConfig.LabelId, functionConfig.DocId, functionConfig.NotificationEmailAddresses)
	if err != nil {
		log.Fatalf("ERROR:Could not create processor:%v\n", err)
	}

	if operation == "" || operation == "invoke" {
		return invoke(processor, functionConfig)
	} else if operation == "selftest" {
		return selftest(processor, functionConfig)
	} else {
		err := fmt.Errorf("invalid operation:%s\n", operation)
		fmt.Printf("ERROR:%v\n", err)
		return err
	}
}

func invoke(processor *common.EmailProcessor, functionConfig FunctionConfig) error {
	executionStatus := processor.LookForAndProcessEmails()

	if executionStatus.ErrString == "" {
		// success!
		fmt.Printf("Successfully processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)

		if b, _ := storeErrorString(noError); b {
			// previous run had an error, so send out notification of all-good
			okstr := "Successful run after previous error(s)"
			processor.AppendToGoogleDocs(okstr)
			processor.SendEmail(functionConfig.NotificationEmailAddresses, "Powerstatus Script OK", okstr)
		}

		if len(executionStatus.WarnMsgs) > 0 {
			warnStr := fmt.Sprintf("Warnings:%v\n", executionStatus.WarnMsgs)
			processor.AppendToGoogleDocs(warnStr)
			processor.SendEmail(functionConfig.NotificationEmailAddresses, "Powerstatus Script Warning", warnStr)
		}
	} else {
		// error!
		errStr := fmt.Sprintf("ERROR:%s\n", executionStatus.ErrString)
		fmt.Printf(errStr)

		if b, _ := storeErrorString(executionStatus.ErrString); b {
			// first error -- i.e., no previous error so send out notification
			processor.AppendToGoogleDocs(errStr)
			processor.SendEmail(functionConfig.NotificationEmailAddresses, "Powerstatus Script Error!", errStr)
			return fmt.Errorf(errStr)
		}
	}
	return nil
}

func selftest(processor *common.EmailProcessor, functionConfig FunctionConfig) error {
	// send myself a gmail with appropriate label set so that the
	// next invoke() will pick it up
	executionStatus := processor.StartSelftest()
	if executionStatus.ErrString == "" {
		// success!
		fmt.Printf("Sent selftest")
		if len(executionStatus.WarnMsgs) > 0 {
			warnStr := fmt.Sprintf("Sending selftest warnings:%v\n", executionStatus.WarnMsgs)
			processor.AppendToGoogleDocs(warnStr)
			processor.SendEmail(functionConfig.NotificationEmailAddresses, "Sending selftest Script Warning", warnStr)
		}
	} else {
		// error!
		errStr := fmt.Sprintf("Sending selftest ERROR:%s\n", executionStatus.ErrString)
		processor.SendEmail(functionConfig.NotificationEmailAddresses, "Sending selftest Script Error!", errStr)
		return fmt.Errorf(errStr)
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
