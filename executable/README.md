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
This program uses two files for security: credentials.json and token.json.

credentials.json is the secret key used for OAuth2 authentication to gmail/docs
token.json is a short-lived bearer token that is automatically updated

token.json is based on the scopes (permissions) required.  If any scope changes, 
or somehow becomes invalid (Google can do that), then 
delete token.json to start the new token.json re-creation process, which is:
1. rm token.json
2. ./run
3. Copy output of ./run to a browser and follow instructions
4. Ignore the "This site can’t be reached" error on the browser, but instead copy everything after "code=" up to "&" 
   and paste that exact string into the waiting ./run program, which reads from stdin and creates token.json
Note: probably need to update the google cloud Datastore when a new vesion of token.json so that the function works
