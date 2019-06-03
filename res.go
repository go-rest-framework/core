package core

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/asaskevich/govalidator"
)

type Errs []ErrorMsg

type ErrorMsg struct {
	Item string `json:"item"`
	Msg  string `json:"msg"`
}

type Response struct {
	Errors Errs        `json:"errors"`
	Data   interface{} `json:"data"`
}

func (e *Errs) Add(item string, msg string) {
	*e = append(*e, ErrorMsg{item, msg})
}

func (r *Response) Make() []byte {
	if len(r.Errors) > 0 {
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

		t := strings.Split(err.Error(), ";")
		for _, v := range t {
			tt := strings.Split(v, ": ")
			r.Errors.Add(tt[0], tt[1])
		}
		return false
	}
	return true
}
