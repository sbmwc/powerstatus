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

	flag.Parse()

	//	fmt.Println("labelId:", *labelIdPtr)
	//	fmt.Println("docIdPtr:", *docIdPtr)
	//	fmt.Println("printAllLabelIds:", *printAllLabelIdsPtr)
	//	fmt.Println("printDocName:", *printDocNamePtr)
	//fmt.Println("tail:", flag.Args())

	client := getHTTPClientUsingFilesystem()

	executionStatus := common.LookForAndProcessEmails(client, *labelIdPtr, *docIdPtr, *printAllLabelIdsPtr, *printDocNamePtr)
	if executionStatus.ErrString != "" {
		fmt.Printf("ERROR:%s\n", executionStatus.ErrString)
		return
	}

	if len(executionStatus.WarnMsgs) > 0 {
		fmt.Printf("WARNINGS:%v\n", executionStatus.WarnMsgs)
	}

	fmt.Printf("Processed %d messages:%v\n", len(executionStatus.MsgIdsProcessed), executionStatus.MsgIdsProcessed)
}
