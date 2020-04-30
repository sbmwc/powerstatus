This is the root project for the SBMWC's powerstatus app.  This is written in go, so you mush
have golang installed on your system.

The software is stored on github under the sbmwc account.  o clone:
$ cd $GOPATH
$ mkdir sbmwc
$ cd sbmwc
$ git clone https://github.com/sbmwc/powerstatus.git

There are two versions: a command-line application and a google cloud function.
Both use the same underlying code, just the invocation is different

directory:
- executable: the main program for the executable (command-line) version
- function: the deployable function in google cloud
- common: the core/common code that is applicable for both the executable and function

