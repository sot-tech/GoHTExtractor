/*
 * BSD-3-Clause
 * Copyright 2020 sot (PR_713, C_rho_272)
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 * 1. Redistributions of source code must retain the above copyright notice,
 * this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 * this list of conditions and the following disclaimer in the documentation and/or
 * other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its contributors
 * may be used to endorse or promote products derived from this software without
 * specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
 * IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT,
 * INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
 * BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA,
 * OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
 * WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY
 * OF SUCH DAMAGE.
 */

package HTExtractor

import (
	"container/list"
	"errors"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var logger = logging.MustGetLogger("sot-te.ch/HTExtractor")

const (
	actGo      = "go"
	actExtract = "extract"
	actFindAll = "findAll"
	actFindFirst = "findFirst"
	actStore   = "store"

	paramSearch = "${search}"
	paramArg    = "${arg}"

	httpProtoString = "http"
)

type ExtractAction struct {
	Action string `json:"action"`
	Param  string `json:"param"`
}

type Extractor struct {
	functions       *list.List
	currentFunction *list.Element
	baseUrl         string
	search          string
	data            map[string][]byte
	lastUrl         string
}

func New(actions []ExtractAction) *Extractor {
	ex := new(Extractor)
	ex.functions = list.New()
	for actIndex := len(actions) - 1; actIndex >= 0; actIndex-- {
		action := actions[actIndex]
		var currentFunc func(string, []byte)bool
		switch action.Action {
		case actGo:
			currentFunc = func(selector string, paramBytes []byte)bool {
				var nextFunc func(string, []byte)bool
				var param string
				if paramBytes != nil {
					param = string(paramBytes)
				}
				if ex.currentFunction != nil {
					ex.currentFunction = ex.currentFunction.Next()
					if ex.currentFunction != nil {
						nextFunc = ex.currentFunction.Value.(func(string, []byte)bool)
					}
				}
				context := strings.ReplaceAll(action.Param, paramArg, param)
				context = strings.ReplaceAll(context, paramSearch, ex.search)
				if strings.Index(param, httpProtoString) != 0 {
					context = ex.baseUrl + context
				}
				if resp, err := http.Get(context); err == nil && resp != nil && resp.StatusCode < 400 {
					ex.lastUrl = context
					if bytes, err := ioutil.ReadAll(resp.Body); err == nil {
						if nextFunc != nil {
							return nextFunc(selector, bytes)
						}
					} else {
						logger.Warningf("Read body error: %v", err)
					}
				} else {
					var errMsg string
					if err != nil {
						errMsg = err.Error()
					} else if resp == nil {
						errMsg = "empty response"
					} else {
						errMsg = resp.Status
					}
					err = errors.New(errMsg)
					logger.Warningf("HTTP error: %s", errMsg)
				}
				return false
			}
		case actExtract:
			currentFunc = func(selector string, param []byte)bool {
				var nextFunc func(string, []byte)bool
				if ex.currentFunction != nil {
					ex.currentFunction = ex.currentFunction.Next()
					if ex.currentFunction != nil {
						nextFunc = ex.currentFunction.Value.(func(string, []byte)bool)
					}
				}
				if param == nil {
					param = make([]byte, 0)
				}
				pattern := strings.ReplaceAll(action.Param, paramSearch, ex.search)
				if reg, err := regexp.Compile("(?s)" + pattern); err == nil {
					//groupNames := reg.SubexpNames()
					matches := reg.FindAllSubmatch(param, -1)
					if matches != nil {
						//TODO: selector
						for _, match := range matches {
							if match != nil && len(match) > 1 {
								if nextFunc != nil {
									tmpF := ex.currentFunction
									if nextFunc("", match[1]) { //FIXME: HERE!
										return true
									}
									ex.currentFunction = tmpF
								}
							}
						}
					}
				} else {
					logger.Warning(err)
				}
				return false
			}
		case actFindAll:
			currentFunc = func(selector string, param []byte)bool {
				var nextFunc func(string, []byte)bool
				if ex.currentFunction != nil {
					ex.currentFunction = ex.currentFunction.Next()
					if ex.currentFunction != nil {
						nextFunc = ex.currentFunction.Value.(func(string, []byte)bool)
					}
				}
				if param == nil {
					param = make([]byte, 0)
				}
				if action.Param == "" {
					if len(param) > 0 && nextFunc != nil {
						return nextFunc(selector, param)
					}
				} else {
					pattern := strings.ReplaceAll(action.Param, paramSearch, ex.search)
					if reg, err := regexp.Compile("(?s)" + pattern); err == nil {
						if matches := reg.FindSubmatch(param); matches != nil && len(matches) > 0 {
							if nextFunc != nil {
								return nextFunc(selector, param)
							}
						}
					} else {
						logger.Warning(err)
					}
				}
				return false
			}
		case actFindFirst:
			currentFunc = func(selector string, param []byte)bool {
				var nextFunc func(string, []byte)bool
				if ex.currentFunction != nil {
					ex.currentFunction = ex.currentFunction.Next()
					if ex.currentFunction != nil {
						nextFunc = ex.currentFunction.Value.(func(string, []byte)bool)
					}
				}
				if param == nil {
					param = make([]byte, 0)
				}
				if action.Param == "" {
					if len(param) > 0 {
						if nextFunc != nil {
							nextFunc(selector, param)
						}
						return true
					}
				} else {
					pattern := strings.ReplaceAll(action.Param, paramSearch, ex.search)
					if reg, err := regexp.Compile("(?s)" + pattern); err == nil {
						if matches := reg.FindSubmatch(param); matches != nil && len(matches) > 0 {
							if nextFunc != nil {
								nextFunc(selector, param)
							}
							return true
						}
					} else {
						logger.Warning(err)
					}
				}
				return false
			}
		case actStore:
			currentFunc = func(selector string, param []byte)bool {
				var nextFunc func(string, []byte)bool
				if ex.currentFunction != nil {
					ex.currentFunction = ex.currentFunction.Next()
					if ex.currentFunction != nil {
						nextFunc = ex.currentFunction.Value.(func(string, []byte)bool)
					}
				}
				if ex.data == nil {
					ex.data = make(map[string][]byte)
				}
				ex.data[selector] = param
				if nextFunc != nil {
					nextFunc(selector, param)
				}
				return false
			}
		}
		ex.functions.PushFront(currentFunc)
	}
	return ex
}

func (ex *Extractor) ExtractData(baseUrl string, search string) (map[string][]byte, string) {
	var res map[string][]byte
	var lastUrl string
	if ex.functions != nil && ex.functions.Len() > 0 {
		ext := Extractor{
			functions:       ex.functions,
			currentFunction: ex.functions.Front(),
			baseUrl:         baseUrl,
			search:          search,
			data:            make(map[string][]byte),
			lastUrl:         "",
		}
		ext.currentFunction.Value.(func(string, []byte))("", nil)
		res, lastUrl = ext.data, ext.lastUrl
	}
	return res, lastUrl
}
