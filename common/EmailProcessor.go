package common

import (
	"encoding/base64"
	"fmt"
	//	"net"
	"net/http"
	//	"net/url"
	"regexp"
	"sort"
	"strings"
	//"syscall"
	"time"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/gmail/v1"
)

// special username for gmail that means "my account" -- in this case sunsetbeachmutualwatercompany@gmail.com
const user string = "sunsetbeachmutualwatercompany@gmail.com" // "me"
const POWER_FAULT_EVENT_NAME string = "Power is Down"
const POWER_RESTORE_EVENT_NAME string = "Power Restore"
const selftestSubject = "powerstatus selftest"

var eventNameRe *regexp.Regexp
var eventTimeRe *regexp.Regexp

func init() {
	// sample event/email from Mission (but in HTML, and may contain extra whitespace/linebreaks
	//////////////////////
	//Mission Communications.	Mission Communications
	//3170 Reps Miller Rd
	//Suite 190
	//Norcross, GA 30071-5403
	//This is a very important alarm message from Mission Communications. It is for SBMWC at Sunset Beach Mutual Water Company / Apex Consulting .
	// 	Unit Name	Sunset Beach
	// 	Message	Mission RTU AC Power Fault
	// 	Time	11 Apr 2020 15:57:34
	// 	Message #	55614
	//
	//Please call toll-free +1.877.991.1911 immediately to acknowledge this message.
	//When prompted, enter event # 55614.
	//
	//Click here to acknowledge this alarm.
	/////////////////////////

	// note, leading '(?s)' means to allow .* to match \n (don't stop at newlines)
	// Directly from Mission is in one long string, however if forwarded, then
	// newlines can be inserted for whatever reason
	eventNameRe = regexp.MustCompile("(?s)Mission RTU AC (.*?)<") // looks for what happened by name--one of POWER_... above
	eventTimeRe = regexp.MustCompile("(?s)Time.*?(\\d+.*?)<")     // looks for the time the event happened
}

type EmailProcessor struct {
	client                     *http.Client
	labelId                    string
	docId                      string
	statusEmailAddresses       string
	notificationEmailAddresses string
	gmailService               *gmail.Service
	docsService                *docs.Service
}

type LabelInfo struct {
	Id   string
	Name string
}

func GetNeededScopes() []string {
	return []string{
		gmail.MailGoogleComScope, docs.DocumentsScope,
	}
}

func NewEmailProcessor(client *http.Client, labelId string, docId string, statusEmailAddresses string, notificationEmailAddresses string) (*EmailProcessor, error) {
	gmailService, err := gmail.New(client)
	if err != nil {
		return nil, err
	}

	docsService, err := docs.New(client)
	if err != nil {
		return nil, err
	}

	return &EmailProcessor{client, labelId, docId, statusEmailAddresses, notificationEmailAddresses, gmailService, docsService}, nil
}

func (processor *EmailProcessor) GetAllLabels() ([]LabelInfo, error) {
	labelsList, err := processor.gmailService.Users.Labels.List(user).Do()
	if err != nil {
		return nil, err
	}

	var result []LabelInfo

	for _, l := range labelsList.Labels {
		result = append(result, LabelInfo{l.Id, l.Name})
	}
	return result, nil
}

func (processor *EmailProcessor) GetDocName() (string, error) {
	doc, err := processor.docsService.Documents.Get(processor.docId).Do()
	if err != nil {
		return "unknown", err
	}
	return doc.Title, nil
}

func (processor *EmailProcessor) AppendToGoogleDocs(str string) error {
	now := time.Now()

	loc, _ := time.LoadLocation("US/Pacific")
	if loc != nil {
		now = now.In(loc)
	}
	formattedTime := now.Format("Mon Jan 2 3:04:05 MST")

	batchUpdateDocRequest := &docs.BatchUpdateDocumentRequest{}
	batchUpdateDocRequest.Requests = append(batchUpdateDocRequest.Requests,
		&docs.Request{
			InsertText: &docs.InsertTextRequest{
				Text:                 fmt.Sprintf("%s:: %s\n", formattedTime, str),
				EndOfSegmentLocation: &docs.EndOfSegmentLocation{},
			},
		})
	_, err := processor.docsService.Documents.BatchUpdate(processor.docId, batchUpdateDocRequest).Do()
	return err
}

func (processor *EmailProcessor) SendEmail(to string, subject string, plaintext string) error {

	header := make(map[string]string)
	header["From"] = "sunsetbeachmutualwatercompany@gmail.com"
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=utf-8"
	header["Content-Transfer-Encoding"] = "base64"

	var sb strings.Builder

	for k, v := range header {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(plaintext)

	gmsg := gmail.Message{
		Raw: base64.RawURLEncoding.EncodeToString([]byte(sb.String())),
	}

	_, err := processor.gmailService.Users.Messages.Send("me", &gmsg).Do()
	if err != nil {
		return err
	}
	//fmt.Printf("Sent message id:%s\n", sentMessage.Id)
	return nil
}

func (processor *EmailProcessor) DeleteEmail(msgId string) {
	// move to trash -- possible to permanently delete a message via Delete()
	processor.gmailService.Users.Messages.Trash("me", msgId).Do()
}

func (processor *EmailProcessor) fetchMsgIds(executionStatus *ExecutionStatus) []string {
	query := "is:unread"
	msgIds := []string{}
	pageToken := ""
	var firstErr error = nil
	for {
		req := processor.gmailService.Users.Messages.List("me").LabelIds(processor.labelId).Q(query)
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			// try one additional time -- we've seen google can return a TCP read error--connectionr reset by peer
			// and I think because the google email service was restarted in between sucessive calls from us, so just try one additional time
			// before giving up.  Note that we are doing this retry for all errors, we probably could trap
			// specific errors as in this, but could not get to work
			//			if urlErr, ok := err.(*url.Error); ok {
			//				switch t := urlErr.Err.(type) {
			//				//https://stackoverflow.com/questions/22761562/portable-way-to-detect-different-kinds-of-network-error-in-golang
			//				case *net.OpError:
			//					network error
			//
			//				case syscall.Errno:
			//		            if t == syscall.ECONNRESET { ...
			//
			if firstErr == nil {
				firstErr = err
				//executionStatus.addWarnMsg(fmt.Sprintf("First time: retrieve msgs from gmail failed, error:%v", firstErr))
				continue
			}

			// we already retried, so this is now an error
			executionStatus.ErrString = fmt.Sprintf("Second time: retrieve msgs from gmail failed, error:%v\n AND \nfirstErr:%v", err, firstErr)
			return nil
		}
		// allow retries to be done.  Consider retry worked once, but we got multiple pages.  Should we really do this?
		firstErr = nil

		for _, msg := range resp.Messages {
			msgIds = append(msgIds, msg.Id)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	// sort message ID in ascending order -- gmail returns possibly in reverse order of arrival
	sort.Strings(msgIds)
	return msgIds
}

func (processor *EmailProcessor) LookForAndProcessEmails() *ExecutionStatus {
	executionStatus := &ExecutionStatus{}

	// get set of unread msgs on label
	msgIds := processor.fetchMsgIds(executionStatus)
	if msgIds == nil {
		return executionStatus
	}

	// read msgs and process one by one
	for _, msgId := range msgIds {
		msg, err := processor.gmailService.Users.Messages.Get(user, msgId).Format("full").Do()
		if err != nil {
			executionStatus.ErrString = fmt.Sprintf("Unable to retrieve message %v: %v", msgId, err)
			return executionStatus
		}

		// see if this is a testing message and if so, handle that
		if isSelftestEmail(msg) {
			processor.AppendToGoogleDocs("Selftest OK")
			if processor.statusEmailAddresses != "" {
				processor.SendEmail(processor.statusEmailAddresses, "Selftest OK", "")
			} else {
				fmt.Printf("Selftest OK\n")
			}
			// delete the original selftest email
			processor.DeleteEmail(msgId)

		} else {
			body := getEmailContent(msg)
			if body == "" {
				executionStatus.addWarnMsg(fmt.Sprintf("no body content for msg:%s\n", msgId))
			} else {
				eventName, eventTime := getEventNameAndTime(body)

				if eventName == "" || eventTime == "" {
					executionStatus.addWarnMsg(fmt.Sprintf("ignoring msgId:%s because could not find eventName (%s) or eventTime (%s) in body %s\n", msgId, eventName, eventTime, body))
				} else if !isValidEventName(eventName) {
					executionStatus.addWarnMsg(fmt.Sprintf("invalid event name received:%s\n", eventName))

				} else {
					docsContent := formatPowerDocsContent(eventName, eventTime)
					err = processor.AppendToGoogleDocs(docsContent)
					if err != nil {
						executionStatus.addWarnMsg(fmt.Sprintf("could not append msg %s to google docs:%v\n", docsContent, err))
						// this is not considered fatal
					}

					emailContent := formatPowerEmailContent(eventName, eventTime)
					err = processor.SendEmail(processor.notificationEmailAddresses, "Power Status Notification", emailContent)
					if err != nil {
						executionStatus.ErrString = fmt.Sprintf("Unable to send email to google groups, content:%s, err:%v\n", emailContent, err)
						return executionStatus
					}
				}
			}

			// mark msg as read (remove UNREAD label)
			msg, err = processor.gmailService.Users.Messages.Modify("me", msgId, &gmail.ModifyMessageRequest{
				RemoveLabelIds: []string{"UNREAD"},
			}).Do()
			if err != nil {
				executionStatus.ErrString = fmt.Sprintf("Unable to mark mesageID %s as UNREDAD, err:%v", msgId, err)
				return executionStatus
			}
		}

		executionStatus.addMsgId(msgId)
	}
	return executionStatus
}

func (processor *EmailProcessor) StartSelftest() *ExecutionStatus {
	executionStatus := &ExecutionStatus{}

	// send an email with labelID set
	err := processor.SendEmail(user, selftestSubject, "")
	if err != nil {
		executionStatus.ErrString = fmt.Sprintf("Unable to send selftest email:%v\n", err)
		return executionStatus
	}

	err = processor.AppendToGoogleDocs("Sent selftest email")
	if err != nil {
		executionStatus.addWarnMsg(fmt.Sprintf("could not append selftest msg to google docs:%v\n", err))
		// this is not considered fatal
	}

	return executionStatus
}

func isValidEventName(eventName string) bool {
	return eventName == POWER_FAULT_EVENT_NAME || eventName == POWER_RESTORE_EVENT_NAME
}

func isSelftestEmail(msg *gmail.Message) bool {
	// look for subject that indicates this is a selftest msg
	msgPart := msg.Payload
	if msgPart != nil {
		msgPartHeaders := msgPart.Headers
		if msgPartHeaders != nil {
			for _, header := range msgPartHeaders {
				if strings.EqualFold(header.Name, "subject") {
					return strings.EqualFold(header.Value, selftestSubject)
				}
			}
		}
	}
	return false
}

func getEventNameAndTime(body string) (string, string) {
	var eventName, eventTime string

	matches := eventNameRe.FindStringSubmatch(body)
	if matches != nil {
		eventName = matches[1]
		//fmt.Printf("eventName:%s\n", eventName)
	}

	matches = eventTimeRe.FindStringSubmatch(body)
	if matches != nil {
		eventTime = matches[1]
		//fmt.Printf("eventTime in email body:%s\n", eventTime)

		// The time in the email body isn't great.  So we'll attempt to convert it
		// to a more user-friendly time format
		loc, _ := time.LoadLocation("US/Pacific")
		parsedTime, err := time.ParseInLocation("_2 Jan 2006 15:04:05", eventTime, loc)
		if err == nil {
			eventTime = parsedTime.Format("Mon Jan 2 3:04:05 PM MST")
		}
	}

	return eventName, eventTime
}

func getContentFromMessagePart(part *gmail.MessagePart) string {
	mimeType := part.MimeType
	//	fmt.Printf("mimeType:%s\n", mimeType)
	//	d, _ := base64.URLEncoding.DecodeString(part.Body.Data)
	//	fmt.Printf("part:%s\n", string(d))

	// mimeType describes the content as in:
	// text/plain: the message BODY only in plain text
	// text/html: the message BODY only in HTML
	// multipart/alternative: will contain two PARTS that are alternatives for each othe, for example:
	//    a text/plain part for the message body in plain text
	//    a text/html part for the message body in html

	if mimeType == "text/html" || mimeType == "text/plain" {
		body := part.Body
		if body == nil {
			return ""
		}
		data, _ := base64.URLEncoding.DecodeString(body.Data)
		return string(data)

	} else if mimeType == "multipart/alternative" {
		// more complicated -- content consists of multiple types, called parts
		// each in their own part section
		for _, part := range part.Parts {
			content := getContentFromMessagePart(part)
			if content != "" {
				return content
			}
		}
	}
	return ""
}

func getEmailContent(msg *gmail.Message) string {
	payload := msg.Payload
	return getContentFromMessagePart(payload)
}

func formatPowerDocsContent(eventName string, eventTime string) string {
	return fmt.Sprintf("EventName:%s;  EventTime:%s", eventName, eventTime)
}

func formatPowerEmailContent(eventName string, eventTime string) string {
	var sb strings.Builder

	sb.WriteString("As of ")
	sb.WriteString(eventTime)
	sb.WriteString(" ")
	if eventName == POWER_FAULT_EVENT_NAME {
		sb.WriteString("AC power has been LOST at the Sunset Beach Mutual Water Company (SBMWC) site.")
	} else if eventName == POWER_RESTORE_EVENT_NAME {
		sb.WriteString("AC power has been RESTORED at the Sunset Beach Mutual Water Company (SBMWC) site.")
	} else {
		sb.WriteString("An unknown event has occurred at the Sunset Beach Mutual Water Company (SBMWC) site.")
	}
	sb.WriteString("\n\r")
	sb.WriteString("--------------")
	sb.WriteString("\n\r")
	sb.WriteString("This is a notification of a power status change at the Sunset Beach Mutual Water Company (SBMWC) site.  " +
		"The pumps that pressurize the water supply use electricity supplied by the grid (PG&E). " +
		"If the power is out, there will be little or no water pressure. This condition will " +
		"continue until the electricity is restored.")
	if eventName == POWER_FAULT_EVENT_NAME {
		sb.WriteString("  There is no time estimate for when power will be restored.")
	}
	sb.WriteString("\n\r")
	sb.WriteString("If you have a life-threatening emergency, please dial 911. " +
		"If you have a water emergency, please view the SBMWC's web site http://sunsetbeachmutualwatercompany.org " +
		"for the water manager's contact information.")

	return sb.String()

}
