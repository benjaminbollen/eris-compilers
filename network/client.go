package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/eris-ltd/eris-compilers/util"

	"github.com/eris-ltd/common/go/common"
	log "github.com/eris-ltd/eris-logger"
)

//todo: Might also need to add in a map of library names to addrs
func BeginCompile(url string, file string, optimize bool, libraries string) (*util.Response, error) {
	common.InitErisDir()
	request, err := util.CreateRequest(file, libraries, optimize)
	if err != nil {
		return nil, err
	}
	//todo: check server for newer version of same files...
	// go through all includes, check if they have changed
	cached := util.CheckCached(request.Includes, request.Language)

	log.WithField("cached?", cached).Debug("Cached Item(s)")

	/*for k, v := range request.Includes {
		log.WithFields(log.Fields{
			"k": k,
			"v": string(v.Script),
		}).Debug("check request loop")
	}*/

	var resp *util.Response
	// if everything is cached, no need for request
	if cached {
		// TODO: need to return all contracts/libs tied to the original src file
		resp, err = util.CachedResponse(request.Includes, request.Language)
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("Could not find cached object, compiling...")
		if url == "" {
			resp = util.Compile(request)
		} else {
			resp, err = requestResponse(request, url)
			if err != nil {
				return nil, err
			}
		}
		resp.CacheNewResponse(*request)
	}

	return resp, nil
}

// send an http request and wait for the response
func requestResponse(req *util.Request, URL string) (*util.Response, error) {
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

	respJ := new(util.Response)
	// read in response body
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, respJ)
	if err != nil {
		log.Errorln("failed to unmarshal", err)
		return nil, err
	}
	return respJ, nil
}
