/*******************************************************************************
 * Copyright 2017 Samsung Electronics All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 *******************************************************************************/
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/edgexfoundry/core-config-seed-go/internal/pkg"
	"github.com/edgexfoundry/core-config-seed-go/internal/pkg/config"
	"github.com/edgexfoundry/core-config-seed-go/pkg/v2/types"
	"github.com/fatih/structs"
	"github.com/pelletier/go-toml"
	"github.com/consulstructure"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/magiconair/properties"
	"gopkg.in/yaml.v2"
)

var Version = "master"

const (
	consulStatusPath = "/v1/agent/self"
	configDefault    = "configuration.toml"
)

// Hook the functions in the other packages for the tests.
var (
	consulDefaultConfig = consulapi.DefaultConfig
	consulNewClient     = consulapi.NewClient
	consulDeleteTree    = (*consulapi.KV).DeleteTree
	consulPut           = (*consulapi.KV).Put
	consulKeys          = (*consulapi.KV).Keys
	httpGet             = http.Get
)


var allowOptions = map[string]string{"name": "", "default": ""}


// END Consul parse
func main() {

	var useConsul bool
	var useProfile string

	flag.BoolVar(&useConsul, "consul", false, "Indicates the service should use consul.")
	flag.BoolVar(&useConsul, "c", false, "Indicates the service should use consul.")
	flag.StringVar(&useProfile, "profile", "", "Specify a profile other than default.")
	flag.StringVar(&useProfile, "p", "", "Specify a profile other than default.")
	flag.Parse()

	// Configuration data for the config-seed service.
	coreConfig := &pkg.CoreConfig{}

	// Load based on configuration need (docker or go)
	err := config.LoadFromFile(coreConfig)
	if err != nil {
		logBeforeTermination(err)
		return
	}

	consulClient, err := getConsulClient(*coreConfig)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	kv := consulClient.KV()

	if coreConfig.IsReset {
		removeStoredConfig(kv)
	}
	// load V2 config files
	loadV2ConfigFromPath(useProfile, *coreConfig, kv)

	// load V1 config files
	loadConfigFromPath(*coreConfig, kv)

	printBanner("./res/banner.txt")
}

// Print a banner.
func printBanner(path string) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(string(b))
}

func logBeforeTermination(err error) {
	fmt.Println(err.Error())
}

func determineConfigFile(profile string) string {
	if profile == "" {
		return configDefault
	}
	return "configuration-" + profile + ".toml"
}

func determineCorrectConfigFile(targetConfig string, fileName string) bool {

	if fileName == targetConfig {
		return true
	}
	return false
}

// Get handle of Consul client using the URL from configuration info.
// Before getting handle, it tries to receive a response from a Consul agent by simple health-check.
func getConsulClient(coreConfig pkg.CoreConfig) (*consulapi.Client, error) {

	consulUrl := coreConfig.ConsulProtocol + "://" + coreConfig.ConsulHost + ":" + strconv.Itoa(coreConfig.ConsulPort)

	// Check the connection to Consul
	fails := 0
	for fails < coreConfig.FailLimit {
		resp, err := httpGet(consulUrl + consulStatusPath)
		if err != nil {
			fmt.Println(err.Error())
			time.Sleep(time.Second * time.Duration(coreConfig.FailWaitTime))
			fails++
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
	}
	if fails >= coreConfig.FailLimit {
		return nil, errors.New("Cannot get connection to Consul")
	}

	// Connect to the Consul Agent
	configTemp := consulDefaultConfig()
	configTemp.Address = consulUrl

	return consulNewClient(configTemp)
}

// Remove all values in Consul K/V store, under the globalprefix which is presents in configuration file.
func removeStoredConfig(kv *consulapi.KV) {
	_, err := consulDeleteTree(kv, pkg.CoreConfiguration.GlobalPrefix, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("All values under the globalPrefix(\"" + pkg.CoreConfiguration.GlobalPrefix + "\") is removed.")
}

// Check if Consul has been configured by trying to get any key that starts with a globalprefix.
func isConfigInitialized(coreConfig pkg.CoreConfig, kv *consulapi.KV) bool {
	keys, _, err := consulKeys(kv, coreConfig.GlobalPrefix, "", nil)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	if len(keys) > 0 {
		fmt.Printf("%s exists! The configuration data has been initialized.\n", coreConfig.GlobalPrefix)
		return true
	}
	fmt.Printf("%s doesn't exist! Start importing configuration data.\n", coreConfig.GlobalPrefix)
	return false
}

// Load a property file(.yaml or .properties) and parse it to a map.
func readPropertyFile(coreConfig pkg.CoreConfig, filePath string) (pkg.ConfigProperties, error) {

	if isTomlExtension(coreConfig, filePath) {
		// Read .toml
		return readTomlFile(filePath)
	} else if isYamlExtension(coreConfig, filePath) {
		// Read .yaml/.yml file
		return readYamlFile(filePath)
	} else {
		// Read .properties file
		return readPropertiesFile(filePath)
	}

}


// V2 Config changes in parsing and loading.
// NOTE a simple inline test so you can read the values out after you push them.
// TODO: change the inline to an external test and run it against the file
func loadV2ConfigFromPath(profile string, coreConfig pkg.CoreConfig, kv *consulapi.KV) {

	err := filepath.Walk(coreConfig.ConfigPathV2, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories & unacceptable property extension
		if info.IsDir() || !isAcceptablePropertyExtensions(coreConfig, info.Name()) {
			return nil
		}

		dir, file := filepath.Split(path)
		configPath, err := filepath.Rel(".", coreConfig.ConfigPathV2)
		if err != nil {
			return err
		}

		// Use correct configFileName
		fileName := determineConfigFile(profile)

		// Skip incorrect configs
		if !determineCorrectConfigFile(fileName, file) {
			return nil
		}

		dir = strings.TrimPrefix(dir, configPath+"/")
		fmt.Println("found config file:", file, "in context", dir)

		// load the ToML file
		config,_ := toml.LoadFile(path)

		// Fetch the map[string]interface{}
		m := config.ToMap()

		// traverse the map and put into KV[]
		kvs, err := traverse("",m)
		if err != nil {
			fmt.Printf("There was an error: %v", err)
		}
		for _, kv := range kvs {
			fmt.Println("v2 consul wrote key", kv.Key, "with value", kv.Value)
		}

		// Put config properties to Consul K/V store.
		prefix := coreConfig.GlobalPrefix + "/" + dir

		// Put config properties to Consul K/V store.
		for _,v := range kvs {
			p := &consulapi.KVPair{Key: prefix + v.Key, Value: []byte(v.Value)}
			if _, err := consulPut(kv, p, nil); err != nil {
				return err
			}
		}

		// TEST make sure we have our values from Consul
		// Let's read the values from Consul K/V store now
		// In our clients we hook this up and we can receive updates from consul changes
		updateCh := make(chan interface{})
		errCh := make(chan error)
		d := &consulstructure.Decoder{
			Target:   &types.EdgeX_Core_Command{},
			Prefix:   "config/EdgeX_Core_Command",
			UpdateCh: updateCh,
		}
		defer d.Close()
		go d.Run()


		// NOTE: Place breakpoint here..change the actual loaded values in consul
		//       They should be pulled via the channel update call and you should see the changes here.
		//       Look at "actual" value for the map[string]interface{}
		var raw interface{}
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("timeout")
		case err := <-errCh:
			fmt.Printf("err: %s", err)
		case raw = <-updateCh:
		}

		actual := raw.(*types.EdgeX_Core_Command)
		if actual== nil {
			fmt.Printf("bad: %#v", actual)
		}

		// END TEST client read from consul

		return nil
	})

	// Special case V2 config parsing here

	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

// V1 Config - Load all config files and put the configuration info to Consul K/V store.
func loadConfigFromPath(coreConfig pkg.CoreConfig, kv *consulapi.KV) {
	err := filepath.Walk(coreConfig.ConfigPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories & unacceptable property extension
		if info.IsDir() || !isAcceptablePropertyExtensions(coreConfig, info.Name()) {
			return nil
		}

		dir, file := filepath.Split(path)
		configPath, err := filepath.Rel(".", coreConfig.ConfigPath)
		if err != nil {
			return err
		}

		dir = strings.TrimPrefix(dir, configPath+"/")
		fmt.Println("found config file:", file, "in context", dir)

		// Parse *.properties
		props, err := readPropertyFile(coreConfig, path)
		if err != nil {
			return err
		}

		// Put config properties to Consul K/V store.
		prefix := coreConfig.GlobalPrefix + "/" + dir
		// here we need to make sure we add all the keys as appropriate
		for k := range props {
			p := &consulapi.KVPair{Key: prefix + k, Value: []byte(props[k])}
			if _, err := consulPut(kv, p, nil); err != nil {
				return err
			}
		}
		return nil
	})

	// Special case V2 config parsing here

	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func isAcceptablePropertyExtensions(coreConfig pkg.CoreConfig, file string) bool {
	for _, v := range coreConfig.AcceptablePropertyExtensions {
		if v == filepath.Ext(file) {
			return true
		}
	}
	return false
}

// Check whether a filename extension is yaml or not.
func isYamlExtension(coreConfig pkg.CoreConfig, file string) bool {
	for _, v := range coreConfig.YamlExtensions {
		if v == filepath.Ext(file) {
			return true
		}
	}
	return false
}

func isTomlExtension(coreConfig pkg.CoreConfig, file string) bool {
	for _, v := range coreConfig.TomlExtensions {
		if v == filepath.Ext(file) {
			return true
		}
	}
	return false
}

//This works for now because our TOML is simply key/value.
//Will not work once we go hierarchical
func readTomlFile(filePath string) (pkg.ConfigProperties, error) {
	configProps := pkg.ConfigProperties{}

	file, err := os.Open(filePath)
	if err != nil {
		return configProps, fmt.Errorf("could not load configuration file (%s): %v", filePath, err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()   // The Line of text were gathering


		if strings.Contains(line, "[") {
			// Parse the text until the next "]"



		}
		if strings.Contains(line, "=") {
			tokens := strings.Split(scanner.Text(), "=")
			configProps[strings.Trim(tokens[0], " '")] = strings.Trim(tokens[1], " '")
		}


	}
	return configProps, nil
}

// Parse a yaml file to a map.
func readYamlFile(filePath string) (pkg.ConfigProperties, error) {

	configProps := pkg.ConfigProperties{}

	contents, err := ioutil.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(contents, configProps); err != nil {
		return nil, err
	}

	m := structs.Map(configProps)

	for key, value := range m {
		m[key] = fmt.Sprint(value)
	}

	return configProps, nil
}

// Parse a properties file to a map.
func readPropertiesFile(filePath string) (pkg.ConfigProperties, error) {

	configProps := pkg.ConfigProperties{}

	props, err := properties.LoadFile(filePath, properties.UTF8)
	if err != nil {
		return nil, err
	}
	configProps = props.Map()

	return configProps, nil
}


// Key/Value pair for parsing
type KV struct {
	Key   string
	Value string
}

// Traverse or walk a map with an optional path start
func traverse(path string, j interface{}) ([]*KV, error) {
	kvs := make([]*KV, 0)

	pathPre := ""
	if path != "" {
		pathPre = path + "/"
	}

	switch j.(type) {
	case []interface{}:
		for sk, sv := range j.([]interface{}) {
			skvs, err := traverse(pathPre+strconv.Itoa(sk), sv)
			if err != nil {
				return nil, err
			}
			kvs = append(kvs, skvs...)
		}
	case map[string]interface{}:
		for sk, sv := range j.(map[string]interface{}) {
			skvs, err := traverse(pathPre+sk, sv)
			if err != nil {
				return nil, err
			}
			kvs = append(kvs, skvs...)
		}
	case int:
		kvs = append(kvs, &KV{Key: path, Value: strconv.Itoa(j.(int))})
	case int64:

		var y int = int(j.(int64))

		kvs = append(kvs, &KV{Key: path, Value: strconv.Itoa(y)})
		//kvs = append(kvs, &KV{Key: path, Value: strconv.FormatInt(j.(int64),64)})
	case float64:
		kvs = append(kvs, &KV{Key: path, Value: strconv.FormatFloat(j.(float64), 'f', -1, 64)})
	case bool:
		kvs = append(kvs, &KV{Key: path, Value: strconv.FormatBool(j.(bool))})
	case nil:
		kvs = append(kvs, &KV{Key: path, Value: ""})
	default:
		kvs = append(kvs, &KV{Key: path, Value: j.(string)})
	}

	return kvs, nil
}