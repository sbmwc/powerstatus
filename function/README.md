This is the google cloud version of the SBMWC powerstatus application.  It is based
on google's cloud functions

To build, use ./build
To deploy pub/sub version, use ./deploy       
To execute pub/sub version (it will execute itself normally though google scheduler), use ./invoke
To see the logs, use ./view-logs

To deploy HTTP version ./http-deploy
To execute HTTP, use ./http-invoke

To see more details of the function or environment, go to https://console.cloud.google.com and log in
as sunsetbeachmutualwatercompany@gmail.com

Requirements to be set in the google datastore:
kind=credentials  name=gmail-credentials   Properties={
    JsonCredentials=<Json encoded credentials>
    JsonToken=<Json encoded token>
}
kind=config  name=function-config  Properties={
    DocId=<docId>
    LabelId=<labelId>
    ErrorAndWarnEmailAddresses=<email where status and warning msgs go to>
}
kind=errors  name=last-error  Properties={
    ErrorString="<none>"
}

credentials/user-credentials can be obtained from executable directory
