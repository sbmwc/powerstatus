module sbmwc/powerstatus/function

go 1.14

require (
	cloud.google.com/go/datastore v1.1.0
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	sbmwc/powerstatus/common v1.0.0
)

replace sbmwc/powerstatus/common v1.0.0 => ../common
