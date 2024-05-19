package main

import (
	"fmt"
)

type MSDPData struct {
	Name  string
	Value interface{}
}

func parseData(data []byte) []MSDPData {
	var result []MSDPData
	var name, value string
	var readingName, readingValue bool
	var containers []interface{}
	var currentContainerName string

	addToContainer := func(containers *[]interface{}, name string, value interface{}) {
		if len(*containers) == 0 {
			result = append(result, MSDPData{Name: name, Value: value})
		} else {
			switch container := (*containers)[len(*containers)-1].(type) {
			case map[string]interface{}:
				container[name] = value
			case *[]interface{}:
				*container = append(*container, value)
			}
		}
	}

	for _, b := range data {
		switch b {
		case 1: // MSDP_VAR
			if readingValue {
				addToContainer(&containers, currentContainerName, value)
				value = ""
			}
			readingName = true
			readingValue = false
			name = ""
		case 2: // MSDP_VAL
			if readingName {
				currentContainerName = name
			}
			readingName = false
			readingValue = true
		case 3: // MSDP_TABLE_OPEN
			fmt.Printf("Table open while var is %s\n", currentContainerName)
			if readingValue {
				if len(value) > 0 {
					addToContainer(&containers, currentContainerName, value)
					value = ""
				}
				readingValue = false
			}
			containers = append(containers, make(map[string]interface{}))
		case 4: // MSDP_TABLE_CLOSE
			if len(containers) == 0 {
				break
			}
			if readingValue {
				if len(value) > 0 {
					addToContainer(&containers, currentContainerName, value)
					value = ""
				}
				readingValue = false
			}
			currentContainer := containers[len(containers)-1]
			containers = containers[:len(containers)-1] // pop from stack
			switch typedContainer := currentContainer.(type) {
			case map[string]interface{}:
				addToContainer(&containers, currentContainerName, typedContainer)
			}
		case 5: // MSDP_ARRAY_OPEN
			if readingValue {
				if len(value) > 0 {
					addToContainer(&containers, name, value)
					value = ""
				}
				readingValue = false
			}
			containers = append(containers, &[]interface{}{})
		case 6: // MSDP_ARRAY_CLOSE
			if len(containers) == 0 {
				break
			}

			if readingValue {
				if len(value) > 0 {
					addToContainer(&containers, currentContainerName, value)
					value = ""
				}
				readingValue = false
			}

			currentContainer := containers[len(containers)-1]
			containers = containers[:len(containers)-1] // pop from stack
			switch typedContainer := currentContainer.(type) {
			case *[]interface{}:
				addToContainer(&containers, currentContainerName, *typedContainer)
			}
		case 255: // END of SB
			// Ensure the last name-value pair is added

		default:
			if readingName {
				name += string(b)
			} else if readingValue {
				value += string(b)
			}
		}
	}

	return result
}

func main() {
	data := []byte{
		1, 71, 82, 79, 85, 80, 2, 5, 2, 3, 1, 110, 97, 109, 101, 2, 75, 101, 110, 115, 104, 111,
		1, 104, 101, 97, 108, 116, 104, 2, 49, 48, 48, 1, 109, 97, 110, 97, 2, 49, 48, 48, 1, 115,
		116, 97, 109, 105, 110, 97, 2, 49, 48, 48, 1, 99, 108, 97, 115, 115, 2, 71, 79, 68, 1, 114,
		97, 99, 101, 2, 111, 116, 104, 101, 114, 1, 110, 112, 99, 2, 112, 99, 1, 112, 111, 115, 105,
		116, 105, 111, 110, 2, 70, 108, 121, 105, 110, 103, 1, 119, 105, 116, 104, 95, 108, 101, 97,
		100, 101, 114, 2, 49, 1, 102, 108, 97, 103, 115, 2, 27, 91, 48, 59, 51, 50, 109, 87, 50, 49,
		57, 1, 119, 105, 116, 104, 95, 121, 111, 117, 2, 49, 1, 105, 115, 95, 108, 101, 97, 100, 101,
		114, 2, 49, 1, 105, 115, 95, 115, 117, 98, 108, 101, 97, 100, 101, 114, 2, 48, 1, 108, 101,
		118, 101, 108, 2, 50, 49, 57, 4, 2, 3, 1, 110, 97, 109, 101, 2, 102, 97, 109, 105, 108, 105,
		97, 114, 1, 104, 101, 97, 108, 116, 104, 2, 49, 48, 48, 1, 109, 97, 110, 97, 2, 49, 48, 48,
		1, 115, 116, 97, 109, 105, 110, 97, 2, 49, 48, 48, 1, 99, 108, 97, 115, 115, 2, 78, 80, 67,
		1, 114, 97, 99, 101, 2, 116, 97, 109, 101, 32, 97, 110, 105, 109, 97, 108, 1, 110, 112, 99,
		2, 110, 112, 99, 1, 112, 111, 115, 105, 116, 105, 111, 110, 2, 83, 116, 97, 110, 100, 105,
		110, 103, 1, 119, 105, 116, 104, 95, 108, 101, 97, 100, 101, 114, 2, 49, 1, 102, 108, 97, 103,
		115, 2, 1, 119, 105, 116, 104, 95, 121, 111, 117, 2, 49, 1, 105, 115, 95, 108, 101, 97, 100,
		101, 114, 2, 48, 1, 105, 115, 95, 115, 117, 98, 108, 101, 97, 100, 101, 114, 2, 48, 1, 108,
		101, 118, 101, 108, 2, 53, 4, 6,
	}

	x := parseData(data)
	fmt.Printf("%+v\n", x)
}
