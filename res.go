package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/asaskevich/govalidator"
)

type Errs []ErrorMsg

type ErrorMsg struct {
	Item string `json:"item"`
	Msg  string `json:"msg"`
}

type Response struct {
	Errors Errs          `json:"errors"`
	Data   interface{}   `json:"data"`
	Count  int64         `json:"count"`
	Req    *http.Request `json:"-"`
}

func (e *Errs) Add(item string, msg string) {
	*e = append(*e, ErrorMsg{item, msg})
}

func (r *Response) Make() []byte {
	if len(r.Errors) > 0 || r.isCheckValidate() {
		r.Data = nil
	}
	res, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return res
}

func (r *Response) IsJsonParseDone(jsn io.Reader) bool {
	decoder := json.NewDecoder(jsn)
	err := decoder.Decode(r.Data)
	if err != nil {
		r.Errors.Add("json", err.Error())
		return false
	}
	return true
}

func (r *Response) IsValidate() bool {
	_, err := govalidator.ValidateStruct(r.Data)
	if err != nil {

		fmt.Println("VALID ERROR:", err.Error())

		t := strings.Split(err.Error(), ";")
		for _, v := range t {
			tt := strings.Split(v, ": ")
			r.Errors.Add(tt[0], tt[1])
		}
		return false
	}
	if r.isCheckValidate() {
		return false
	}
	return true
}

func (r *Response) isCheckValidate() bool {
	if r.Req.Header.Get("isValidate") == "1" {
		return true
	}
	return false
}
