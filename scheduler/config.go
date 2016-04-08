/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

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

package scheduler

// default configuration values
const (
	defaultListenAddr           string = "127.0.0.1"
	defaultListenPort           int    = 8081
	defaultWorkManagerQueueSize uint   = 25
	defaultWorkManagerPoolSize  uint   = 4
)

// holds the configuration passed in through the SNAP config file
type Config struct {
	ListenAddr           string `json:"listen_addr,omitempty"yaml:"listen_addr,omitempty"`
	ListenPort           int    `json:"listen_port,omitempty"yaml:"listen_port,omitempty"`
	WorkManagerQueueSize uint   `json:"work_manager_queue_size,omitempty"yaml:"work_manager_queue_size,omitempty"`
	WorkManagerPoolSize  uint   `json:"work_manager_pool_size,omitempty"yaml:"work_manager_pool_size,omitempty"`
}

// get the default snapd configuration
func GetDefaultConfig() *Config {
	return &Config{
		ListenAddr:           defaultListenAddr,
		ListenPort:           defaultListenPort,
		WorkManagerQueueSize: defaultWorkManagerQueueSize,
		WorkManagerPoolSize:  defaultWorkManagerPoolSize,
	}
}
