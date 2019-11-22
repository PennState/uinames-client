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

type opt func(v *url.Values) error

func intOpt(k string, i int) opt {
	return func(v *url.Values) error {
		v.Set(k, fmt.Sprintf("%v", i))
		return nil
	}
}

func Amount(amount int) opt {
	return func(v *url.Values) error {
		if amount < 1 || amount > 500 {
			return errors.New(amountErrorMsg)
		}
		a := fmt.Sprintf("%v", amount)
		v.Set(string(amountKey), a)
		return nil
	}
}

func ExtraData() opt {
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

func Gender(gender gender) opt {
	return func(v *url.Values) error {
		v.Set(string(genderKey), string(gender))
		return nil
	}
}

func MaximumLength(max int) opt {
	return intOpt(string(maxLenKey), max)
}

func MinimumLength(min int) opt {
	return intOpt(string(minLenKey), min)
}

func Region(region string) opt {
	return func(v *url.Values) error {
		v.Set(string(regionKey), region)
		return nil
	}
}

type Request struct {
	*http.Request
}

func NewRequest(opts ...opt) (*Request, error) {
	// URL is known to be valid at this point
	u, _ := url.Parse(URL)
	v := url.Values{}
	for _, opt := range opts {
		err := opt(&v)
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

type Response struct {
	Name       string     `json:"name"`
	Surname    string     `json:"surname"`
	Gender     string     `json:"gender"`
	Region     string     `json:"region"`
	Age        int        `json:"age"`
	Title      string     `json:"title"`
	Phone      string     `json:"phone"`
	Birthdate  Birthdate  `json:"birthday"`
	Email      string     `json:"email"`
	Password   string     `json:"password"`
	CreditCard CreditCard `json:"credit_card"`
	Photo      string     `json:"photo"` // TODO: convert to URL
}

type Birthdate time.Time

func (b Birthdate) UnmarshalJSON(data []byte) error {
	dob := struct {
		raw int64 `json:"raw"`
	}{
		raw: 0,
	}
	err := json.Unmarshal(data, &dob)
	if err != nil {
		return err
	}
	return nil
}

type CreditCard struct {
	Expiration string `json:"expiration"`
	Number     string `json:"number"`
	Pin        int    `json:"pin"`
	Security   int    `json:"security"`
}

type UINamesError struct {
	Message    string `json:"error"`
	Status     string `json:"-"`
	StatusCode int    `json:"-"`
}

func (e UINamesError) Error() string {
	return e.Status + " - " + e.Message
}

func unmarshalError(resp *http.Response) error {
	e := UINamesError{
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
