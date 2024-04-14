/*
Copyright 2024 Sourav Patnaik.

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

package controller

type History struct {
	Data string `json:"data"`
}

type Secret struct {
	Name         string    `json:"name"`
	Created      string    `json:"created"`
	LastModified string    `json:"last_modified"`
	History      []History `json:"history"`
	Type         string    `json:"type"`
}

type Namespace struct {
	Name    string   `json:"name"`
	Secrets []Secret `json:"secrets"`
}

type ConfigData struct {
	Namespaces []Namespace `json:"namespaces"`
}
