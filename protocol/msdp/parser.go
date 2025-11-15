// Package msdp provides a parser for the Mud Server Data Protocol (MSDP) as defined
// in the Telnet sub-negotiation format. It parses the raw byte slice of a Telnet
// MSDP segment (IAC SB MSDP [data] IAC SE) into a Go map[string]interface{} where
// values can be strings, []interface{} (arrays), or map[string]interface{} (tables).
//
// The parser handles nested tables and arrays, as well as special "list" commands
// (e.g., REPORT) where multiple VALs follow a single VAR, treating the value as an array
// of strings.
package msdp

import (
	"errors"
	"fmt"
	"strings"
)

const (
	IAC         = 255
	SB          = 250
	SE          = 240
	MSDP        = 69
	VAR         = 1
	VAL         = 2
	TABLE_OPEN  = 3
	TABLE_CLOSE = 4
	ARRAY_OPEN  = 5
	ARRAY_CLOSE = 6
)

// ParseMSDP parses the provided byte slice as an MSDP Telnet sub-negotiation segment.
// It validates the structure (IAC SB MSDP [data] IAC SE) and extracts the data into
// a map[string]interface{}, handling variables, values, tables, arrays, and list commands.
func ParseMSDP(input []byte) (map[string]interface{}, error) {
	if len(input) < 4 {
		return nil, errors.New("input too short for MSDP segment")
	}
	if input[0] != IAC || input[1] != SB || input[2] != MSDP {
		return nil, errors.New("not a valid MSDP sub-negotiation start")
	}
	last := len(input) - 1
	if last < 1 || input[last] != SE || input[last-1] != IAC {
		return nil, errors.New("invalid MSDP sub-negotiation end")
	}
	data := input[3 : len(input)-2] // Extract MSDP data bytes
	return parseData(data)
}

// parseData parses the raw MSDP data bytes into the object.
func parseData(data []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	i := 0
	for i < len(data) {
		// Stop if we hit IAC (255) - this marks the end of the MSDP data
		if data[i] == 255 {
			break
		}
		if data[i] != VAR {
			return nil, fmt.Errorf("expected VAR (1) at position %d, got %d", i, data[i])
		}
		i++ // Skip VAR
		varName, err := readString(data, &i)
		if err != nil {
			return nil, fmt.Errorf("error reading variable name at %d: %w", i, err)
		}
		if i >= len(data) || data[i] == 255 {
			m[varName] = "" // Empty value if no VAL or hit IAC
			break
		}
		if data[i] != VAL {
			m[varName] = "" // No value provided (VAR without VAL)
			continue
		}
		i++ // Skip VAL
		
		// Check if VAL is immediately followed by another control byte (empty value)
		if i >= len(data) || data[i] == 255 {
			m[varName] = "" // Empty value
			break
		}
		if data[i] == VAR || data[i] == TABLE_CLOSE || data[i] == ARRAY_CLOSE {
			// Empty value - VAL followed immediately by another control byte
			m[varName] = ""
			// Don't advance i, let the next iteration handle the VAR
			continue
		}
		
		value, err := parseValue(data, &i)
		if err != nil {
			return nil, fmt.Errorf("error parsing value for %s at %d: %w", varName, i, err)
		}
		// Check for chained VALs (list commands like REPORT)
		if i < len(data) && data[i] == VAL {
			arr := []interface{}{value}
			for i < len(data) && data[i] == VAL {
				i++ // Skip VAL
				// Check for empty chained value
				if i >= len(data) || data[i] == 255 || data[i] == VAR {
					// Empty value in chain
					arr = append(arr, "")
					if data[i] == VAR {
						// Don't advance, let next iteration handle VAR
						break
					}
					if data[i] == 255 {
						break
					}
					continue
				}
				nextVal, err := parseValue(data, &i)
				if err != nil {
					return nil, fmt.Errorf("error parsing chained value at %d: %w", i, err)
				}
				arr = append(arr, nextVal)
			}
			m[varName] = arr
		} else {
			m[varName] = value
		}
		// Stop if we hit IAC after parsing the value
		if i < len(data) && data[i] == 255 {
			break
		}
	}
	return m, nil
}

// parseValue parses a single value starting at the current position.
// It can be a string, table, or array.
func parseValue(data []byte, i *int) (interface{}, error) {
	if *i >= len(data) {
		return "", errors.New("unexpected end of data in value")
	}
	b := data[*i]
	switch b {
	case TABLE_OPEN:
		*i++
		return parseTable(data, i)
	case ARRAY_OPEN:
		*i++
		return parseArray(data, i)
	default:
		s, err := readString(data, i)
		return s, err
	}
}

// parseTable parses a table (map) until TABLE_CLOSE.
func parseTable(data []byte, i *int) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	for *i < len(data) && data[*i] != TABLE_CLOSE {
		if data[*i] != VAR {
			return nil, fmt.Errorf("expected VAR in table at %d", *i)
		}
		*i++
		key, err := readString(data, i)
		if err != nil {
			return nil, fmt.Errorf("error reading table key at %d: %w", *i, err)
		}
		if *i >= len(data) || data[*i] != VAL {
			return nil, errors.New("expected VAL in table")
		}
		*i++
		val, err := parseValue(data, i)
		if err != nil {
			return nil, fmt.Errorf("error parsing table value at %d: %w", *i, err)
		}
		m[key] = val
	}
	if *i < len(data) && data[*i] == TABLE_CLOSE {
		*i++
		return m, nil
	}
	return nil, errors.New("unclosed table")
}

// parseArray parses an array until ARRAY_CLOSE.
func parseArray(data []byte, i *int) ([]interface{}, error) {
	arr := make([]interface{}, 0, 4)
	for *i < len(data) && data[*i] != ARRAY_CLOSE {
		if data[*i] != VAL {
			return nil, fmt.Errorf("expected VAL in array at %d", *i)
		}
		*i++
		val, err := parseValue(data, i)
		if err != nil {
			return nil, fmt.Errorf("error parsing array value at %d: %w", *i, err)
		}
		arr = append(arr, val)
	}
	if *i < len(data) && data[*i] == ARRAY_CLOSE {
		*i++
		return arr, nil
	}
	return nil, errors.New("unclosed array")
}

// readString reads bytes until a control byte (0,1,2,3,4,5,6,255) or end.
func readString(data []byte, i *int) (string, error) {
	start := *i
	for *i < len(data) {
		b := data[*i]
		if b == 0 || b == 1 || b == 2 || b == 3 || b == 4 || b == 5 || b == 6 || b == 255 {
			break
		}
		*i++
	}
	if *i == start {
		return "", nil // Empty string
	}
	// Check for forbidden chars (though spec forbids them)
	seg := data[start:*i]
	for _, b := range seg {
		if b == 0 || b == 255 {
			return "", fmt.Errorf("forbidden byte 0x%02x in string", b)
		}
	}
	return string(seg), nil
}

// PrettyPrint returns a human-readable representation of the MSDP object for debugging.
func PrettyPrint(m map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("{")
	for k, v := range m {
		sb.WriteString(fmt.Sprintf("\n  %s: ", k))
		sb.WriteString(formatValue(v, 2))
		sb.WriteString(",")
	}
	sb.WriteString("\n}")
	return sb.String()
}

func formatValue(v interface{}, indent int) string {
	ind := strings.Repeat(" ", indent)
	switch vv := v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", vv)
	case map[string]interface{}:
		var sb strings.Builder
		sb.WriteString("{\n")
		for k, val := range vv {
			sb.WriteString(fmt.Sprintf("%s  %s: %s,\n", ind, k, formatValue(val, indent+2)))
		}
		sb.WriteString(fmt.Sprintf("%s}", ind))
		return sb.String()
	case []interface{}:
		var sb strings.Builder
		sb.WriteString("[\n")
		for _, val := range vv {
			sb.WriteString(fmt.Sprintf("%s  %s,\n", ind, formatValue(val, indent+2)))
		}
		sb.WriteString(fmt.Sprintf("%s]", ind))
		return sb.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

