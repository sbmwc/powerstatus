package function

import (
	"fmt"
	"net/http"
	"sbmwc/powerstatus/common"
)

// InvokeEmailProcessor is an HTTP Cloud Function with a request parameter.
// Note this is a GET and probably should be a POST (or PUT), since it has side effects
func HttpProcessor(w http.ResponseWriter, r *http.Request) {

	client := getHTTPClientUsingDB()
	options := getOptionsFromDB()

	executionStatus := common.LookForAndProcessEmails(client, options.LabelId, options.DocId, false, false)
	if executionStatus.ErrString != "" {
		fmt.Fprintf(w, "ERROR:%s\n  Google Docs should be updated and error email sent", executionStatus.ErrString)

		common.AppendToGoogleDocs(common.DocsService(), options.DocId, "ERROR:"+executionStatus.ErrString)
		common.SendEmail(common.GmailService(), "jim@planeshavings.com", "Powerstatus Script Error!", "ERROR:"+executionStatus.ErrString)
		return

	}

	fmt.Fprintf(w, "Successfully processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)
}
