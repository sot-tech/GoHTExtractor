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
	"fmt"
	"github.com/op/go-logging"
	"io/ioutil"
	"regexp"
	"sot-te.ch/HTExtractor"
)

type Config struct{
	Actions []HTExtractor.ExtractAction `json:"actions"`
	BaseUrl string `json:"baseurl"`
}

var s = `drwx------ 1 sot  sot     168 окт 28 15:11  .adobe
drwxrwxr-x 1 sot  sot       0 дек  6 09:39  .android
-rw-rw-r-- 1 sot  sot     561 дек 21  2018  .anyconnect
drwxrwxr-x 1 sot  sot     792 янв  3  2019  .arduino15
drwxrwxr-x 1 sot  sot     920 ноя 29 10:38  .audacity-data
-rw-rw-r-- 1 sot  sot      15 фев 15  2018  .bash_aliases`

var r = `(?s)<div class="main_title">.*?<span>(?P<name>.*?)<\/span>|<div class="release_reln"><span>(?P<lat_name>.*?)<\/span><\/div>`

func main() {
	flag.Parse()
	args := flag.Args()
	logger := logging.MustGetLogger("main")
	if len(args) == 0 {
		logger.Fatal("config file not set")
	}
	confData, err := ioutil.ReadFile(args[0])
	re := regexp.MustCompile(r)
	groupNames := re.SubexpNames()
	for matchNum, match := range re.FindAllSubmatch(confData, -1) {
		for groupIdx, group := range match {
			name := groupNames[groupIdx]
			if name == "" {
				name = "*"
			}
			fmt.Printf("#%d text: '%s', group: '%s'\n", matchNum, string(group), name)
		}
	}
	if err == nil {
		conf := new(Config)
		if err = json.Unmarshal(confData, conf); err == nil {
			ex := HTExtractor.New(conf.Actions)
			ex.ExtractData(conf.BaseUrl, "")
		}
	}
	logger.Fatal(err)
}
