// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-content-service/server"
	"github.com/RedHatInsights/insights-content-service/types"
)

// APIRequest is a request to api to use in AssertAPIRequest
//
// (required) Method is an http method
// (required) Endpoint is an endpoint without api prefix
// EndpointArgs are the arguments to pass to endpoint template (leave empty if endpoint is not a template)
// Body is a string body (leave empty to not send)
// UserID is a user id for methods requiring user id (leave empty to not use it)
// XRHIdentity is an authentication token (leave empty to not use it)
// AuthorizationToken is an authentication token (leave empty to not use it)
type APIRequest struct {
	Method             string
	Endpoint           string
	EndpointArgs       []interface{}
	Body               string
	UserID             types.UserID
	XRHIdentity        string
	AuthorizationToken string
}

// APIResponse is an expected api response to use in AssertAPIRequest
//
// StatusCode is an expected http status code (leave empty to not check for status code)
// Body is an expected body string (leave empty to not check for body)
// BodyChecker is a custom body checker function (leave empty to use default one - CheckResponseBodyJSON)
type APIResponse struct {
	StatusCode  int
	Body        string
	BodyChecker func(t testing.TB, expected, got string)
	Headers     map[string]string
}

// DefaultServerConfig is a default config used by AssertAPIRequest
var DefaultServerConfig = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	UseHTTPS:    false,
}

// AssertAPIRequest creates new server
// (which you can keep nil so it will be created automatically)
// and provided serverConfig(you can leave it empty to use the default one)
// sends api request and checks api response (see docs for APIRequest and APIResponse)
func AssertAPIRequest(
	t testing.TB,
	serverConfig *server.Configuration,
	request *APIRequest,
	expectedResponse *APIResponse,
) {
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}

	groupsData := make(map[string]groups.Group)
	groupsData["foo"] = groups.Group{
		Name:        "group name: foo",
		Description: "group description: foo",
		Tags:        []string{"tag1", "tag2"},
	}
	groupsData["bar"] = groups.Group{
		Name:        "group name: bar",
		Description: "group description: bar",
		Tags:        []string{"tag3", "tag4"},
	}
	testServer := server.New(*serverConfig, groupsData)

	url := server.MakeURLToEndpoint(serverConfig.APIPrefix, request.Endpoint, request.EndpointArgs...)

	req := makeRequest(t, request, url)

	response := ExecuteRequest(testServer, req, serverConfig).Result()

	if len(expectedResponse.Headers) != 0 {
		checkResponseHeaders(t, expectedResponse.Headers, response.Header)
	}
	if expectedResponse.StatusCode != 0 {
		assert.Equal(t, expectedResponse.StatusCode, response.StatusCode, "Expected different status code")
	}
	if expectedResponse.BodyChecker != nil {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		FailOnError(t, err)

		expectedResponse.BodyChecker(t, expectedResponse.Body, string(bodyBytes))
	} else if len(expectedResponse.Body) != 0 {
		CheckResponseBodyJSON(t, expectedResponse.Body, response.Body)
	}
}

func makeRequest(t testing.TB, request *APIRequest, url string) *http.Request {
	req, err := http.NewRequest(request.Method, url, strings.NewReader(request.Body))
	FailOnError(t, err)

	if len(request.AuthorizationToken) != 0 {
		req.Header.Set("Authorization", request.AuthorizationToken)
	}

	return req
}

// ExecuteRequest executes http request on a testServer
func ExecuteRequest(testServer *server.HTTPServer, req *http.Request, config *server.Configuration) *httptest.ResponseRecorder {
	router := testServer.Initialize(config.Address)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

// CheckResponseBodyJSON checks if body is the same json as in expected
// (ignores whitespaces, newlines, etc)
// also validates both expected and body to be a valid json
func CheckResponseBodyJSON(t testing.TB, expectedJSON string, body io.ReadCloser) {
	result, err := ioutil.ReadAll(body)
	FailOnError(t, err)

	AssertStringsAreEqualJSON(t, expectedJSON, string(result))
}

// checkResponseHeaders checks if headers are the same as in expected
func checkResponseHeaders(t testing.TB, expectedHeaders map[string]string, actualHeaders http.Header) {
	for key, value := range expectedHeaders {
		assert.Equal(t, value, actualHeaders.Get(key), "Expected different headers")
	}
}
