package kallisti

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/perlsaiyan/zif/protocol/msdp"
)

const IAC = byte(255)
const SB = byte(250)
const SE = byte(240)

const MSDP = byte(69)
const MSDP_VAR = byte(1)
const MSDP_VAL = byte(2)
const MSDP_TABLE_OPEN = byte(3)
const MSDP_TABLE_CLOSE = byte(4)
const MSDP_ARRAY_OPEN = byte(5)
const MSDP_ARRAY_CLOSE = byte(6)

type MSDPHandler struct {
	Data map[string]interface{}
	c    net.Conn
	mu   sync.RWMutex // Protects Data map from concurrent access
}

func NewMSDP() *MSDPHandler {
	return &MSDPHandler{
		Data: make(map[string]interface{}),
	}
}

func (m MSDPHandler) SendMSDP(s string) {
	m.c.Write(MSDPMessage(
		[]byte{IAC, SB, m.OptionCode()},
		[]byte{MSDP_VAR}, []byte("SEND"), []byte{MSDP_VAL}, []byte(s),
		[]byte{IAC, SE}))
}

func (m MSDPHandler) OptionCode() byte {
	return MSDP
}

func (m MSDPHandler) HandleDo(conn net.Conn) {
	fmt.Printf("Do here")
}

func MSDPMessage(input ...[]byte) []byte {
	var msg []byte
	for _, v := range input {
		msg = append(msg, v...)
	}
	return msg
}

func (m MSDPHandler) HandleWill(c net.Conn) {
	msg := MSDPMessage(
		[]byte{IAC, SB, m.OptionCode()},
		[]byte{MSDP_VAR}, []byte("LIST"), []byte{MSDP_VAL}, []byte("REPORTABLE_VARIABLES"),
		[]byte{IAC, SE})
	c.Write(msg)
}

func (m *MSDPHandler) HandleSB(conn net.Conn, b []byte) {
	// Construct full MSDP segment (IAC SB MSDP [data] IAC SE)
	// b already starts with MSDP byte and may end with IAC (255)
	// If b ends with IAC, we just need to add SE; otherwise add IAC SE
	fullSegment := append([]byte{IAC, SB}, b...)
	if len(fullSegment) > 0 && fullSegment[len(fullSegment)-1] == IAC {
		// Already has IAC, just add SE
		fullSegment = append(fullSegment, SE)
	} else {
		// No IAC at end, add IAC SE
		fullSegment = append(fullSegment, IAC, SE)
	}

	// Parse the MSDP data using the new parser
	parsed, err := msdp.ParseMSDP(fullSegment)
	if err != nil {
		log.Printf("Error parsing MSDP: %v", err)
		return
	}

	// Check if REPORTABLE_VARIABLES is in the newly parsed data (not just in existing map)
	// This ensures we only send REPORT when we actually receive REPORTABLE_VARIABLES
	var reportablesList []string
	if reportables, ok := parsed["REPORTABLE_VARIABLES"]; ok {
		switch v := reportables.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					reportablesList = append(reportablesList, str)
				}
			}
		case []string:
			reportablesList = v
		}
	}

	// Merge parsed data into our Data map
	m.mu.Lock()
	for k, v := range parsed {
		m.Data[k] = v
	}
	m.mu.Unlock()

	// Handle REPORTABLE_VARIABLES - send REPORT request for all variables
	// Only send if we actually received REPORTABLE_VARIABLES in this message
	if len(reportablesList) > 0 {
		msg := MSDPMessage([]byte{IAC, SB, m.OptionCode()}, []byte{MSDP_VAR}, []byte("REPORT"))
		for _, msdpvar := range reportablesList {
			msg = MSDPMessage(msg, []byte{MSDP_VAL}, []byte(msdpvar))
		}
		msg = MSDPMessage(msg, []byte{IAC, SE})
		log.Printf("Got reportables, sending request for %d variables", len(reportablesList))
		conn.Write(msg)
	}
}

// GetString retrieves a string value from the MSDP data
func (m *MSDPHandler) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.Data[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case []interface{}:
			// If it's an array, return empty string or first element as string?
			return ""
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// GetInt retrieves an integer value from the MSDP data
func (m *MSDPHandler) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.Data[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetBool retrieves a boolean value from the MSDP data
// Handles "1"/"0" string conversions and actual boolean values
func (m *MSDPHandler) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.Data[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "1" || v == "true" || v == "True" || v == "TRUE"
		case int:
			return v != 0
		}
	}
	return false
}

// GetArray retrieves an array value from the MSDP data
func (m *MSDPHandler) GetArray(key string) []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.Data[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			return arr
		}
	}
	return nil
}

// GetTable retrieves a table (map) value from the MSDP data
func (m *MSDPHandler) GetTable(key string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.Data[key]; ok {
		if table, ok := val.(map[string]interface{}); ok {
			return table
		}
	}
	return nil
}

// GetAllData returns a copy of all MSDP data for safe iteration
func (m *MSDPHandler) GetAllData() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Create a copy to avoid holding the lock during iteration
	result := make(map[string]interface{}, len(m.Data))
	for k, v := range m.Data {
		result[k] = v
	}
	return result
}
