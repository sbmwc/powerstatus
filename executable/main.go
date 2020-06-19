package main

import (
	"flag"
	"fmt"
	"sbmwc/powerstatus/common"
)

func main() {
	labelIdPtr := flag.String("labelId", "Label_7600509641061721086", "The gmail label ID (not name) to monitor for unread emails")
	docIdPtr := flag.String("docId", "1xKGTENqUHUPI_CEfX8h9DtyvSf-mMDJFKcwa8Sy8AKA", "The google docs ID (not name) to log things")
	printAllLabelIdsPtr := flag.Bool("printLabelIds", false, "Print all labelIDs that are currently created in the gmail account")
	printDocNamePtr := flag.Bool("printDocName", false, "Print the google doc name associated with docId")
	selftestPtr := flag.Bool("selftest", false, "Send a selftest email to sunsetbeachmututalwatercompany@gmail.com")
	notificationEmailAddresses := flag.String("notificationEmailAddresses", "sunsetbeachmutualwatercompany@gmail.com", "Email address(es) where to send notifications")

	flag.Parse()

	//	fmt.Println("labelId:", *labelIdPtr)
	//	fmt.Println("docIdPtr:", *docIdPtr)
	//	fmt.Println("printAllLabelIds:", *printAllLabelIdsPtr)
	//	fmt.Println("printDocName:", *printDocNamePtr)
	//fmt.Println("tail:", flag.Args())

	client := getHTTPClientUsingFilesystem()

	processor, err := common.NewEmailProcessor(client, *labelIdPtr, *docIdPtr, "", *notificationEmailAddresses)
	if err != nil {
		fmt.Printf("ERROR:Could not create processor:%v\n", err)
		return
	}

	if *printAllLabelIdsPtr {
		labelsInfo, err := processor.GetAllLabels()
		if err != nil {
			fmt.Printf("Could not get labels:%v\n", err)
		} else {
			for _, l := range labelsInfo {
				fmt.Printf("Label Name:%s, LabelId:%s\n", l.Name, l.Id)
			}
		}
		return
	}

	if *printDocNamePtr {
		if *docIdPtr == "" {
			fmt.Printf("Missing docId")
		} else {
			name, err := processor.GetDocName()
			if err != nil {
				fmt.Printf("Could not get doc name:%v\n", err)
			} else {
				fmt.Printf("Doc name:%s\n", name)
			}
		}
		return
	}

	if *selftestPtr {
		executionStatus := processor.StartSelftest()
		if executionStatus.ErrString != "" {
			fmt.Printf("ERROR:%s\n", executionStatus.ErrString)
			return
		}

		if len(executionStatus.WarnMsgs) > 0 {
			fmt.Printf("WARNINGS:%v\n", executionStatus.WarnMsgs)
		}

		fmt.Printf("Sent selftest email message\n")
		return
	}

	executionStatus := processor.LookForAndProcessEmails()
	if executionStatus.ErrString != "" {
		fmt.Printf("ERROR:%s\n", executionStatus.ErrString)
		return
	}

	if len(executionStatus.WarnMsgs) > 0 {
		fmt.Printf("WARNINGS:%v\n", executionStatus.WarnMsgs)
	}

	fmt.Printf("Processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)
}
