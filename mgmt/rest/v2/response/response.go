/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2017 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/negroni"
)

func Write(code int, body interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; version=2")
	w.Header().Set("Version", "beta")

	if !w.(negroni.ResponseWriter).Written() {
		w.WriteHeader(code)
	}

	if body != nil {
		j, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			logrus.Fatalln(err)
		}
		j = bytes.Replace(j, []byte("\\u0026"), []byte("&"), -1)
		fmt.Fprint(w, string(j))
	}
}
