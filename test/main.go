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

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"sot-te.ch/HTExtractor"
)

type Config struct {
	Actions        []HTExtractor.ExtractAction `json:"actions"`
	StackLimit     uint64                      `json:"stackLimit"`
	IterationLimit uint64                      `json:"iterationLimit"`
	BaseUrl        string                      `json:"baseurl"`
	Search         string                      `json:"search"`
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Config file not set")
	}
	confData, err := ioutil.ReadFile(args[0])
	if err == nil {
		conf := new(Config)
		if err = json.Unmarshal(confData, conf); err == nil {
			ex := HTExtractor.New()
			ex.StackLimit = conf.StackLimit
			ex.IterationLimit = conf.IterationLimit
			if err = ex.Compile(conf.Actions); err == nil {
				var data map[string][]byte
				data, err = ex.ExtractData(conf.BaseUrl, conf.Search)
				for k, v := range data {
					var s string
					if len(v) > 2048 {
						s = "LONG VALUE"
					} else {
						s = string(v)
					}
					log.Printf("%s: %s\n", k, s)
				}
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
