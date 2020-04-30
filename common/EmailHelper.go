package common

import (
	"encoding/base64"
	"google.golang.org/api/gmail/v1"
	"strings"
)

func SendEmail(gmailService *gmail.Service, to string, subject string, content string) error {

	header := make(map[string]string)
	header["From"] = "sunsetbeachmutualwatercompany@gmail.com"
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	var sb strings.Builder

	for k, v := range header {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}
	sb.WriteString("\n\r")
	sb.WriteString(content)

	gmsg := gmail.Message{
		Raw: base64.RawURLEncoding.EncodeToString([]byte(sb.String())),
	}

	_, err := gmailService.Users.Messages.Send("me", &gmsg).Do()
	if err != nil {
		return err
	}
	//fmt.Printf("Sent message id:%s\n", sentMessage.Id)
	return nil
}
