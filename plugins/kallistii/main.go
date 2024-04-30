package main

import "log"

func F() {
	log.Printf("Plugin executed")
}

/*
	// In order to load a plugin, do something like this:

	plugin, err := plugin.Open("main.so")
	if err != nil {
		panic(err)
	}

	// lookup exported function "F"
	fn, err := plugin.Lookup("F")
	if err != nil {
		panic(err)
	}
	fn.(func())()
*/
