// Copyright © 2018 Nicolas Surleraux <nsurleraux@gmai.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"mikrodock-cli/cmd"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
)

func main() {
	dir, _ := homedir.Dir()
	mikroPath := path.Join(dir, ".mikrodock")
	if _, err := os.Stat(mikroPath); os.IsNotExist(err) {
		modeInt := int(0777)
		errMkdir := os.Mkdir(mikroPath, os.FileMode(modeInt))
		if errMkdir != nil {
			fmt.Fprintf(os.Stderr, "Cannot create .mikrodock directory in your home : %s\r\n", errMkdir.Error())
			os.Exit(1)
		}
		fmt.Printf("Config directory created inside your home\r\n")
	}
	cmd.Execute()
}
