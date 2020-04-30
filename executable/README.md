This is the executable version of the SBMWC powerstatus application

To build, use ./build
To run, use ./run

run options:
-h for help
-labelId <labelId to use>   to use the non-default label id
-docId <docId to use>       to use the non-default doc id
-printAllLabelIds           to print all available label ids and their names
-printDocName               to print the name of the doc associated with docId


Security/Tokens
This program uses two files for security: credentials.json and tokens.json.

credentials.json is the secret key used for OAuth2 authentication to gmail/docs
tokens.json is a short-lived bearer token that is automatically updated

tokens.json is based on the scopes (permissions) required.  If any scope changes, 
delete tokens.json to start the new tokens.json re-creation process, which involves
accepting the new security screens.  Note: probably need to update the google
cloud Datastore when a new vesion of tokens.json so that the function works
