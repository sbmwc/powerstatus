package common

import (
	"fmt"
	"google.golang.org/api/docs/v1"
	"time"
)

func DocName(docsService *docs.Service, docId string) (string, error) {
	doc, err := docsService.Documents.Get(docId).Do()
	if err != nil {
		return "unknown", err
	}
	return doc.Title, nil
}

func AppendToGoogleDocs(docsService *docs.Service, docId string, str string) error {

	now := time.Now()

	loc, _ := time.LoadLocation("US/Pacific")
	if loc != nil {
		now = now.In(loc)
	}
	formattedTime := now.Format("Mon Jan 2 15:04:05 MST")

	batchUpdateDocRequest := &docs.BatchUpdateDocumentRequest{}
	batchUpdateDocRequest.Requests = append(batchUpdateDocRequest.Requests,
		&docs.Request{
			InsertText: &docs.InsertTextRequest{
				Text:                 fmt.Sprintf("%s:: %s\n", formattedTime, str),
				EndOfSegmentLocation: &docs.EndOfSegmentLocation{},
			},
		})
	_, err := docsService.Documents.BatchUpdate(docId, batchUpdateDocRequest).Do()
	return err
}
