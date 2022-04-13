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
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	actGo        = "go"
	actExtract   = "extract"
	actFindAll   = "findAll"
	actFindFirst = "findFirst"
	actStore     = "store"

	paramSearch   = "${search}"
	paramArg      = "${arg}"
	paramSelector = "${selector}"

	httpProtoString = "http"
)

type EFunc func(string, string, []byte) error

type ExtractAction struct {
	Action   string `json:"action"`
	Param    string `json:"param"`
	function EFunc
}

type Extractor struct {
	Actions           map[string]EFunc
	StackLimit        uint64
	IterationLimit    uint64
	stackLevel        uint64
	functions         *list.List
	currentFuncStruct *list.Element
	baseUrl           string
	search            string
	data              map[string][]byte
	stop              bool
}

func (ex *Extractor) NextFunc(selector string, paramBytes []byte) error {
	var err error
	if ex.StackLimit == 0 || ex.stackLevel < ex.StackLimit {
		var nextFunc EFunc
		var nextFuncParam string
		if ex.currentFuncStruct != nil {
			ex.currentFuncStruct = ex.currentFuncStruct.Next()
			if ex.currentFuncStruct != nil {
				nextFuncStruct := ex.currentFuncStruct.Value.(ExtractAction)
				nextFunc = nextFuncStruct.function
				nextFuncParam = nextFuncStruct.Param
			}
		}
		if nextFunc != nil {
			if err = nextFunc(nextFuncParam, selector, paramBytes); err != nil {
				ex.stop = true
			}
		}
	} else {
		err = errors.New("stack limit reached: " + strconv.FormatUint(ex.stackLevel, 10))
	}
	return err
}

func (ex *Extractor) SetFunction(action string, function EFunc) (EFunc, error) {
	if ex.Actions == nil {
		ex.Actions = make(map[string]EFunc)
	}
	if function == nil {
		return nil, errors.New("function cannot be nil")
	} else {
		prevFunction := ex.Actions[action]
		ex.Actions[action] = function
		return prevFunction, nil
	}
}

func (ex *Extractor) goF(funcParam, selector string, paramBytes []byte) error {
	var err error
	var param string
	if paramBytes != nil {
		param = string(paramBytes)
	}
	context := strings.ReplaceAll(funcParam, paramArg, param)
	context = strings.ReplaceAll(context, paramSelector, selector)
	context = strings.ReplaceAll(context, paramSearch, ex.search)
	if strings.Index(param, httpProtoString) != 0 {
		context = ex.baseUrl + context
	}
	var resp *http.Response
	if resp, err = http.Get(context); err == nil {
		resp.Close = true
		defer resp.Body.Close()
		if resp.StatusCode < 400 {
			var bytes []byte
			if bytes, err = ioutil.ReadAll(resp.Body); err == nil {
				err = ex.NextFunc(selector, bytes)
			}
		} else {
			err = errors.New("url " + context + " HTTP code: " + resp.Status)
		}
	}
	return err
}

func (ex *Extractor) extractF(functionParam, selector string, param []byte) error {
	var err error
	if param == nil {
		param = make([]byte, 0)
	}
	pattern := strings.ReplaceAll(functionParam, paramSearch, ex.search)
	pattern = strings.ReplaceAll(pattern, paramSelector, selector)
	var reg *regexp.Regexp
	if reg, err = regexp.Compile("(?s)" + pattern); err == nil {
		groupNames := reg.SubexpNames()
		extracted := make(map[string][][]byte)
		for _, match := range reg.FindAllSubmatch(param, -1) {
			if len(match) > 1 {
				for groupIdx, group := range match[1:] {
					groupIdx++
					name := groupNames[groupIdx]
					if name == "" {
						name = strconv.FormatInt(int64(groupIdx), 10)
					}
					if len(group) > 0 {
						bytesa := extracted[name]
						if bytesa == nil {
							bytesa = make([][]byte, 0)
						}
						extracted[name] = append(bytesa, group)
					}
				}
			}
		}
		var i uint64 = 0
		for k, va := range extracted {
			for _, v := range va {
				if ex.stop {
					break
				}
				if ex.IterationLimit == 0 || i < ex.IterationLimit {
					tmpF := ex.currentFuncStruct
					if err = ex.NextFunc(k, v); err != nil {
						ex.stop = true
					}
					ex.currentFuncStruct = tmpF
					i++
				} else {
					return errors.New("iteration limit reached: " + strconv.FormatUint(i, 10))
				}
			}
		}

	} else {
		ex.stop = true
	}
	return err
}

func (ex *Extractor) findF(functionParam, selector string, param []byte, breakIfFound bool) error {
	var err error
	if param == nil {
		param = make([]byte, 0)
	}
	if functionParam == "" {
		if len(param) > 0 {
			err = ex.NextFunc(selector, param)
			if breakIfFound {
				ex.stop = true
			}
		}
	} else {
		pattern := strings.ReplaceAll(functionParam, paramSearch, ex.search)
		pattern = strings.ReplaceAll(pattern, paramSelector, selector)
		var reg *regexp.Regexp
		if reg, err = regexp.Compile("(?s)" + pattern); err == nil {
			if matches := reg.FindSubmatch(param); len(matches) > 0 {
				err = ex.NextFunc(selector, param)
				if breakIfFound {
					ex.stop = true
				}
			}
		}
	}
	return err
}

func (ex *Extractor) findAllF(functionParam, selector string, param []byte) error {
	return ex.findF(functionParam, selector, param, false)
}

func (ex *Extractor) findFirstF(functionParam, selector string, param []byte) error {
	return ex.findF(functionParam, selector, param, true)
}

func (ex *Extractor) storeF(functionParam, selector string, param []byte) error {
	if ex.data == nil {
		ex.data = make(map[string][]byte)
	}
	ex.data[functionParam+selector] = param
	return ex.NextFunc(selector, param)
}

func New() *Extractor {
	ex := new(Extractor)
	ex.Actions = map[string]EFunc{
		actGo:        ex.goF,
		actExtract:   ex.extractF,
		actFindAll:   ex.findAllF,
		actFindFirst: ex.findFirstF,
		actStore:     ex.storeF,
	}
	return ex
}

func (ex *Extractor) Compile(actions []ExtractAction) error {
	var err error
	ex.functions = list.New()
	for actIndex := len(actions) - 1; actIndex >= 0; actIndex-- {
		action := actions[actIndex]
		if currentFunc := ex.Actions[action.Action]; currentFunc == nil {
			err = errors.New("function for action '" + action.Action + "' not set")
			break
		} else {
			action.function = currentFunc
			ex.functions.PushFront(action)
		}
	}
	return err
}

func (ex *Extractor) ExtractDataWithSelector(baseUrl, search, initSelector string) (map[string][]byte, error) {
	var err error
	var res map[string][]byte
	if ex.functions != nil && ex.functions.Len() > 0 {
		ex.currentFuncStruct = ex.functions.Front()
		ex.baseUrl = baseUrl
		ex.search = search
		ex.data = make(map[string][]byte)
		ex.stop = false
		ex.stackLevel = 0
		if ex.currentFuncStruct != nil {
			funcStruct := ex.currentFuncStruct.Value.(ExtractAction)
			if funcStruct.function != nil {
				err = funcStruct.function(funcStruct.Param, initSelector, nil)
			}
		}
		res = ex.data
		ex.data = nil
	}
	return res, err
}

func (ex *Extractor) ExtractData(baseUrl, search string) (map[string][]byte, error) {
	return ex.ExtractDataWithSelector(baseUrl, search, "")
}
