package function

import (
	"context"
	"fmt"
	"sbmwc/powerstatus/common"
)

var statusEmailTo string = "jim@planeshavings.com"

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// This is the method invoked by pub/sub
func PubSubProcessor(ctx context.Context, m PubSubMessage) error {
	//	name := string(m.Data)
	//	fmt.Printf("name:%s\n", name)   // name may be "" if nothing set in scheduler

	client := getHTTPClientUsingDB()
	options := getOptionsFromDB()

	executionStatus := common.LookForAndProcessEmails(client, options.LabelId, options.DocId, false, false)

	if executionStatus.ErrString == "" {
		// success!
		fmt.Printf("Successfully processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)

		if b, _ := storeErrorString(NoError()); b {
			// previous run had an error, so send out notification of all-good
			okstr := "Successful run after previous error(s)"
			common.AppendToGoogleDocs(common.DocsService(), options.DocId, okstr)
			common.SendEmail(common.GmailService(), statusEmailTo, "Powerstatus Script OK", okstr)
		}

		if len(executionStatus.WarnMsgs) > 0 {
			warnStr := fmt.Sprintf("Warnings:%v\n", executionStatus.WarnMsgs)
			common.AppendToGoogleDocs(common.DocsService(), options.DocId, warnStr)
			common.SendEmail(common.GmailService(), statusEmailTo, "Powerstatus Script Warning", warnStr)
		}
	} else {
		// error!
		fmt.Printf("ERROR:%s\n", executionStatus.ErrString)

		if b, _ := storeErrorString(executionStatus.ErrString); b {
			// first error -- i.e., no previous error so send out notification
			common.AppendToGoogleDocs(common.DocsService(), options.DocId, "ERROR:"+executionStatus.ErrString)
			common.SendEmail(common.GmailService(), statusEmailTo, "Powerstatus Script Error!", "ERROR:"+executionStatus.ErrString)
		}
	}
	return nil
}
