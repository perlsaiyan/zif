package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)


type Config struct {
	Session Session `yaml:"session"`
 
}

type Session struct {
	Hostname string `yaml:"hostname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func GetConfig() *Config {

	var c Config
	if _, err := os.Stat("conf.yaml"); errors.Is(err, os.ErrNotExist) {
		yamlData, err := yaml.Marshal(&c)

		if err != nil {
			fmt.Printf("Error while Marshaling. %v", err)
		}

		filename :=  "conf.yaml"
		err = ioutil.WriteFile(filename, yamlData, 0600)
		if err != nil {
			fmt.Printf("Unable to write %s\n",filename)
			panic("Unable to generate default config")
		}

		fmt.Printf("Sample config: %+v\n",c)
		fmt.Printf("Dropped sample config in conf.yaml.  Edit and rerun.\n")
		os.Exit(0)
	} else {

	yamlFile, err := os.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err #%v", err)
	}

	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	}

	return &c
}



