package uinames

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	URL                     = "https://uinames.com/api/"
	amountErrorMsg          = "amount must be between 1 and 500 (inclusive)"
	nilResponseBodyErrorMsg = "found nil HTTP response entity body"
)

type queryParamKey string

const (
	amountKey    queryParamKey = "amount"
	extraDataKey queryParamKey = "ext"
	genderKey    queryParamKey = "gender"
	regionKey    queryParamKey = "region"
	maxLenKey    queryParamKey = "maxlen"
	minLenKey    queryParamKey = "minlen"
)

// Opt is a function (normally acccessed via a closure) that adds query
// parameters to the HTTP request.
type Opt func(v *url.Values) error

func intOpt(k string, i int) Opt {
	return func(v *url.Values) error {
		v.Set(k, fmt.Sprintf("%v", i))
		return nil
	}
}

// Amount is a request option that sets the count of randomized names
// being requested from the API.  Valid amounts are 1 through 500
// (inclusive).
func Amount(amount int) Opt {
	return func(v *url.Values) error {
		if amount < 1 || amount > 500 {
			return errors.New(amountErrorMsg)
		}
		a := fmt.Sprintf("%v", amount)
		v.Set(string(amountKey), a)
		return nil
	}
}

// ExtraData is a request option that indicates full identity data should
// be returned.  When this option is not included, only Name, SurName,
// Gender and Region are returned (all other fields will be either nil or
// set to their zero values.)
func ExtraData() Opt {
	return func(v *url.Values) error {
		v.Set(string(extraDataKey), "")
		return nil
	}
}

type gender string

const (
	Female gender = "female"
	Male   gender = "male"
)

// Gender is a request option that indicates that only female or male
// identities should be return.
func Gender(gender gender) Opt {
	return func(v *url.Values) error {
		v.Set(string(genderKey), string(gender))
		return nil
	}
}

// MaximumLength is a request option that specifies the maximum number
// of characters expected in the returned name.
func MaximumLength(max int) Opt {
	return intOpt(string(maxLenKey), max)
}

// MinimumLength is a request option that specifies the minimum number
// of characters expected in the returned name.
func MinimumLength(min int) Opt {
	return intOpt(string(minLenKey), min)
}

// Region is a request option that specifies that returned identities
// should only be from the requested geographic region.
func Region(region string) Opt {
	return func(v *url.Values) error {
		v.Set(string(regionKey), region)
		return nil
	}
}

type Request struct {
	*http.Request
}

// NewRequest creates an HTTP request based on the included request
// options.  Requests can be used multiple times by calling the Get()
// function but there is no provision for rate limiting the request
// as required by the uinames API.
func NewRequest(opts ...Opt) (*Request, error) {
	// URL is known to be valid at this point
	u, _ := url.Parse(URL)
	v := url.Values{}
	for _, Opt := range opts {
		err := Opt(&v)
		if err != nil {
			return nil, err
		}
	}
	u.RawQuery = v.Encode()
	hr, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return &Request{
		Request: hr,
	}, nil
}

// Get returns an array of identities returned from the uinames API as
// specified by the Request.
func (r *Request) Get() ([]Response, error) {
	return r.get(&http.Client{})
}

func (r *Request) get(cl *http.Client) ([]Response, error) {
	rl := []Response{}
	hr, err := (cl).Do(r.Request)
	if err != nil {
		return rl, err
	}
	if hr.StatusCode != 200 {
		return rl, unmarshalError(hr)
	}
	body, err := getResponseEntityBody(hr)
	if err != nil {
		return rl, err
	}
	if len(body) > 0 && body[0] == '[' {
		err = json.Unmarshal(body, &rl)
		return rl, err
	}
	ri := Response{}
	err = json.Unmarshal(body, &ri)
	return append(rl, ri), err
}

// Response contains an individual identity returned from the uinames
// API.  If the ExtraData request option is not included, most of this
// struct's fields will be empty.
type Response struct {
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Gender     string `json:"gender"`
	Region     string `json:"region"`
	Age        int    `json:"age"`
	Title      string `json:"title"`
	Phone      string `json:"phone"`
	Birthdate  time.Time
	Email      string     `json:"email"`
	Password   string     `json:"password"`
	CreditCard CreditCard `json:"credit_card"`
	Photo      *url.URL
}

// UnmarshalJSON implements https://golang.org/pkg/encoding/json/#Unmarshaler
func (r *Response) UnmarshalJSON(data []byte) error {
	type Birthday struct {
		DMY string `json:"dmy"`
		MDY string `json:"mdy"`
		Raw int    `json:"raw"`
	}

	type Alias Response
	alias := &struct {
		*Alias
		Birthdate Birthday `json:"birthday"`
		Photo     string   `json:"photo"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	dob, err := time.Parse("01/02/2006", alias.Birthdate.MDY)
	if err != nil {
		return err
	}
	r.Birthdate = dob

	photo, err := url.Parse(alias.Photo)
	if err != nil {
		return err
	}
	r.Photo = photo

	return nil
}

// CreditCard encapsulates set of fields commonly associated with a credit
// or debit card.
type CreditCard struct {
	Expiration string `json:"expiration"`
	Number     string `json:"number"`
	Pin        int    `json:"pin"`
	Security   int    `json:"security"`
}

// Error is an error that encapsulates both the HTTP status and the
// message returned from the uinames API.
type Error struct {
	Message    string `json:"error"`
	Status     string `json:"-"`
	StatusCode int    `json:"-"`
}

// Error implements https://golang.org/pkg/builtin/#error
func (e Error) Error() string {
	return e.Status + " - " + e.Message
}

func unmarshalError(resp *http.Response) error {
	e := Error{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
	data, err := getResponseEntityBody(resp)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &e)
	if err != nil {
		return err
	}
	return e
}

func getResponseEntityBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return []byte(""), errors.New(nilResponseBodyErrorMsg)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), err
	}
	return data, nil
}
