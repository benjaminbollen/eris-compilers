package compilers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// Compile request object
type Request struct {
	ScriptName string            `json:name"`
	Language   string            `json:"language"`
	Script     []byte            `json:"script"`    // source code file bytes
	Includes   map[string][]byte `json:"includes"`  // filename => source code file bytes
	Libraries  string            `json:"libraries"` // string of libName:LibAddr separated by comma
}

// Compile response object
type ResponseItem struct {
	Objectname string `json:"objectname"`
	Bytecode   []byte `json:"bytecode"`
	ABI        string `json:"abi"` // json encoded
}

type Response struct {
	Objects []ResponseItem `json:"objects"`
	Error   string         `json:"error"`
}

// Proxy request object.
// A proxy request must contain a source.
// If the source is a literal (rather than filename),
// ProxyReq.Literal must be set to true and ProxyReq.Language must be provided
type ProxyReq struct {
	Source    string `json:"source"`
	Literal   bool   `json:"literal"`
	Language  string `json:"language"`
	Libraries string `json:"libraries"` // string of libName:LibAddr separated by comma
}

type ProxyRes struct {
	Bytecode string `json:"bytecode"`
	ABI      string `json:"abi"` // json encoded abi struct
	Error    string `json:"error"`
}

// New Request object from script and map of include files
func NewRequest(script []byte, includes map[string][]byte, lang string, libs string) *Request {
	if includes == nil {
		includes = make(map[string][]byte)
	}
	req := &Request{
		Script:    script,
		Includes:  includes,
		Language:  lang,
		Libraries: libs,
	}
	return req
}

// New response object from bytecode and an error
func NewResponse(objectname string, bytecode []byte, abi string, err error) *Response {
	e := ""
	if err != nil {
		e = err.Error()
	}

	respItem := ResponseItem{
		Objectname: objectname,
		Bytecode:   bytecode,
		ABI:        abi}

	respItemArray := make([]ResponseItem, 1)
	respItemArray[0] = respItem

	return &Response{
		Objects: respItemArray,
		Error:   e,
	}
}

func NewProxyResponse(bytecode []byte, abi string, err error) *ProxyRes {
	e := ""
	if err != nil {
		e = err.Error()
	}
	script := ""
	if bytecode != nil {
		script = hex.EncodeToString(bytecode)
	}
	return &ProxyRes{
		Bytecode: script,
		ABI:      abi,
		Error:    e,
	}
}

// send an http request and wait for the response
func requestResponse(req *Request) (*Response, error) {
	lang := req.Language
	URL := Languages[lang].URL
	// log.Debugf("Lang & URL for request =>\t%s:%s\n", URL, lang)
	// make request
	reqJ, err := json.Marshal(req)
	if err != nil {
		log.Errorln("failed to marshal req obj", err)
		return nil, err
	}
	httpreq, err := http.NewRequest("POST", URL, bytes.NewBuffer(reqJ))
	if err != nil {
		log.Errorln("failed to compose request:", err)
		return nil, err
	}
	httpreq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		log.Errorln("failed to send HTTP request", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	respJ := new(Response)
	// read in response body
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, respJ)
	if err != nil {
		log.Errorln("failed to unmarshal", err)
		return nil, err
	}
	return respJ, nil
}

func printRequest(req *Request) {
	fmt.Println("SCRIPT:", string(req.Script))
	for k, v := range req.Includes {
		fmt.Println("include:", k)
		fmt.Println("SCRIPT:", string(v))
	}
}
