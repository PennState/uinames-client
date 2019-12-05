package uinames

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
  Request testing
*/

func TestAmountWithInvalidValues(t *testing.T) {
	tests := []struct {
		Name  string
		Value int
	}{
		{"Too low", -1},
		{"Too high", 501},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			opt := Amount(test.Value)
			err := opt(&url.Values{})
			require.Error(t, err)
			assert.Equal(t, amountErrorMsg, err.Error())
		})
	}
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		Name string
		Opts []Opt
		URL  string
	}{
		{"No opts", []Opt{}, URL},
		{"All opts", []Opt{
			Amount(5),
			ExtraData(),
			Gender(Female),
			MaximumLength(32),
			MinimumLength(10),
			Region("Germany"),
		},
			URL + "?amount=5&ext=&gender=female&maxlen=32&minlen=10&region=Germany"},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			req, err := NewRequest(test.Opts...)
			require.NoError(t, err)
			assert.Equal(t, test.URL, req.Request.URL.String())
		})
	}
}

const errorOptMsg = "I can't work like this"

func errorOpt(t *testing.T) Opt {
	t.Helper()
	return func(v *url.Values) error {
		return errors.New(errorOptMsg)
	}
}

func TestNewRequestWithErroringOpt(t *testing.T) {
	opts := []Opt{errorOpt(t)}
	_, err := NewRequest(opts...)
	require.Error(t, err)
	assert.Equal(t, errorOptMsg, err.Error())
}

/*
  Response testing
*/

type ResponseRoundTripper struct {
	StatusCode int
	Body       io.ReadCloser
}

func (rt ResponseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: rt.StatusCode,
		Status:     http.StatusText(rt.StatusCode),
		Body:       rt.Body,
	}, nil
}

func assertResponse(t assert.TestingT, object interface{}, msgsAndArgs ...interface{}) bool {
	assert := assert.New(t)
	rl, ok := object.([]Response)
	if !ok || len(rl) < 1 {
		assert.FailNow("assertResponse requires a non-empty array of Response objects")
		return false
	}
	ri := rl[0]
	return assert.NotEmpty(ri.Name) &&
		assert.NotEmpty(ri.Surname) &&
		assert.NotEmpty(ri.Gender) &&
		assert.NotEmpty(ri.Region) &&
		assert.NotZero(ri.Age) &&
		assert.NotEmpty(ri.Title) &&
		assert.NotEmpty(ri.Phone) &&
		assert.NotZero(ri.Birthdate) &&
		assert.NotEmpty(ri.Email) &&
		assert.NotEmpty(ri.Password) &&
		assert.NotEmpty(ri.CreditCard.Expiration) &&
		assert.NotEmpty(ri.CreditCard.Number) &&
		assert.NotZero(ri.CreditCard.Pin) &&
		assert.NotZero(ri.CreditCard.Security) &&
		assert.NotEmpty(ri.Photo)
}

func assertError(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	err, ok := object.(Error)
	if !ok {
		assert.FailNow(t, "assertError requires a non-nil UINamesError object")
	}
	return assert.Equal(t, "Bad Request - Region or language not found", err.Error())
}

func TestGet(t *testing.T) {
	tests := []struct {
		Name         string
		StatusCode   int
		BodyFilePath string
		AssertFunc   assert.ValueAssertionFunc
	}{
		{"Item response", 200, "uinames-item.json", assertResponse},
		{"List response", 200, "uinames-list.json", assertResponse},
		{"Error response", 400, "uinames-error.json", assertError},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			body, err := os.Open("testdata/" + test.BodyFilePath)
			require.NoError(t, err)
			cl := &http.Client{
				Transport: ResponseRoundTripper{
					StatusCode: test.StatusCode,
					Body:       body,
				},
			}
			req, err := NewRequest()
			require.NoError(t, err)
			resp, err := req.get(cl)
			if err != nil {
				assertError(t, err)
				return
			}
			assertResponse(t, resp)
		})
	}
}

type errorReader struct {
	err error
}

func ErrorReader(err error) io.Reader {
	return errorReader{err: err}
}

func (er errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

func TestUnmarshalError(t *testing.T) {
	errorFromUINamesMsgPrefix := `{"error":"`
	errorFromUINamesMsg := "Region or language not found"
	errorFromUINamesMsgSuffix := `"}`
	errorFromUINamesJSON := errorFromUINamesMsgPrefix + errorFromUINamesMsg + errorFromUINamesMsgSuffix
	errorFromUINames := ioutil.NopCloser(strings.NewReader(errorFromUINamesJSON))

	errorFromReadAllMsg := "I can't possibly work in this environment"
	errorFromReadAllErr := errors.New(errorFromReadAllMsg)
	errorFromReadAll := ioutil.NopCloser(ErrorReader(errorFromReadAllErr))

	errorFromUnmarshalMsg := "invalid character 'C' looking for beginning of value"
	errorFromUnmarshalJSON := "Clearly this is not JSON"
	errorFromUnmarshal := ioutil.NopCloser(strings.NewReader(errorFromUnmarshalJSON))

	tests := []struct {
		Name string
		Body io.ReadCloser
		Msg  string
	}{
		{"Error from UINames", errorFromUINames, http.StatusText(http.StatusBadRequest) + " - " + errorFromUINamesMsg},
		{"Nil HTTP body", nil, nilResponseBodyErrorMsg},
		{"Error from ReadAll", errorFromReadAll, errorFromReadAllMsg},
		{"Error from Unmarshal", errorFromUnmarshal, errorFromUnmarshalMsg},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			hr := &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       test.Body,
			}
			err := unmarshalError(hr)
			require.Error(t, err)
			assert.Equal(t, test.Msg, err.Error())
		})
	}
}
