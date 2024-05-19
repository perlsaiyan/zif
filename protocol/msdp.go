package kallisti

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"
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

var MSDP_FIELDS = map[string]MSDPFieldDefinition{
	"ACCOUNT_NAME":         {"AccountName", "string"},
	"ALIGNMENT":            {"Alignment", "int"},
	"ANSI_COLORS":          {"AnsiColors", "integer"},
	"AREA_MINLEVEL":        {"AreaMinLevel", "integer"},
	"CHARACTER_NAME":       {"CharacterName", "string"},
	"CLIENT_VERSION":       {"ClientVersion", "string"},
	"COMMANDS":             {"Commands", "list"},
	"GROUP":                {"Group", "grouplist"},
	"REPORTABLE_VARIABLES": {"Reportables", "list"},
	"ROOM_NAME":            {"RoomName", "string"},
	"ROOM_VNUM":            {"RoomVnum", "string"},
	"ROOM_WEATHER":         {"RoomWeather", "string"},
	"SERVER_ID":            {"Server_ID", "string"},
	"SERVER_TIME":          {"ServerTime", "integer"},
	"SNIPPET_VERSION":      {"SnippetVersion", "string"},
	"UPTIME":               {"Uptime", "integer"},
}

type MSDPFieldDefinition struct {
	Field string
	Type  string
}

type MSDPHandler struct {
	Server_ID          string
	Group              [9]GroupMember
	Commands           []string
	Reportables        []string
	AccountName        string
	Uptime             int
	RoomWeather        string
	CharacterName      string
	ServerTime         int
	SnippetVersion     string
	Affects            []Affect
	Alignment          int
	Experience         int
	ExperienceMax      int
	ExperienceTNL      int
	Health             int
	HealthMax          int
	Level              int
	Race               string
	Class              string
	Mana               int
	ManaMax            int
	Wimpy              int
	Practice           int
	Gold               int
	BankGold           int
	Stamina            int
	StaminaMax         int
	Hitroll            int
	Damroll            int
	AC                 int
	Str                int
	Int                int
	Wis                int
	Dex                int
	Con                int
	Luk                int
	StrPerm            int
	IntPerm            int
	WisPerm            int
	DexPerm            int
	ConPerm            int
	LukPerm            int
	StrMax             int
	IntMax             int
	WisMax             int
	DexMax             int
	ConMax             int
	QPoints            int
	Position           string
	WhoFlags           string
	CombatStance       string
	MountName          string
	MountHealth        int
	MountStamina       int
	MountStaminaMax    int
	Ranged             bool
	Prompt             string
	Wield              string
	Hold               string
	Shield             string
	Quickdraw          string
	Hunger             int
	Thirst             int
	Bardsong           string
	OpponentHealth     int
	OpponentHealthMax  int
	OpponentStamina    int
	OpponentStaminaMax int
	OpponentLevel      int
	OpponentName       string
	OpponentNumber     int
	AreaName           string
	AreaMinLevel       int
	AreaMaxLevel       int
	RoomExits          struct{}
	RoomName           string
	RoomTerrain        string
	RoomVnum           string
	WorldTime          int
	ClientID           string
	ClientVersion      string
	PluginID           string
	AnsiColors         int
	Xterm256Colors     int
	UTF8               int
	Sound              int
	MXP                int
	ParagonLevel       int
	HeroPoints         int
	HeroPointsTNL      int
	NoblePoints        int
	NoblePointsTNL     int
	PCInZone           bool
	PCInRoom           bool
	GroupLevel         int
	Equipment          string
	Queue              int
	RemortLevels       int
	RemortLapsTotal    int

	c net.Conn
}

type Affect struct {
}
type GroupMember struct {
	Class        string
	Flags        string
	Health       int
	Is_Leader    bool
	Is_Subleader bool
	Level        int
	Name         string
	Position     string
	Stamina      int
	With_Leader  bool
	With_You     bool
	Mana         int
	NPC          bool
	Race         string
}

func NewMSDP() *MSDPHandler {
	return &MSDPHandler{}
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

func readString(b []byte) (string, []byte) {
	idx := 0

	for i := 0; i < len(b); i++ {
		if b[i] == MSDP_VAR ||
			b[i] == MSDP_VAL ||
			b[i] == MSDP_TABLE_OPEN ||
			b[i] == MSDP_TABLE_CLOSE ||
			b[i] == MSDP_ARRAY_OPEN ||
			b[i] == MSDP_ARRAY_CLOSE ||
			b[i] == 255 {
			idx = i - 1
			break
		}
		idx = i
	}
	idx += 1

	val := string(b[:idx])
	b = b[idx:]
	return val, b
}

func readInteger(b []byte) (int, []byte) {
	var tmp string
	tmp, b = readString(b)
	var val int
	val, _ = strconv.Atoi(tmp)
	return val, b
}

func readArray(b []byte) ([]string, []byte) {
	var obj []string
	for b[0] != MSDP_ARRAY_CLOSE {
		b = b[1:]
		if b[0] == MSDP_TABLE_OPEN {
			log.Printf("TABLE OPEN\n")
			var x interface{}
			x, b = readTable(b)
			log.Printf("TABLE %v\n", x)
		} else {

			var str string
			str, b = readString(b)
			log.Printf("ARRAY ELEMENT %v\n", str)
			obj = append(obj, str)
		}
	}
	b = b[1:]
	log.Printf("Read Array: %+v", obj)
	return obj, b
}

func readTable(b []byte) (map[string]interface{}, []byte) {
	res := make(map[string]interface{})
	var idx string
	for b[0] != MSDP_TABLE_CLOSE {
		switch b[0] {
		case MSDP_VAR:
			b = b[1:]
			idx, b = readString(b)
		case MSDP_VAL:
			b = b[1:]
			res[idx], b = readString(b)
		default:
			fmt.Printf("Default case in readTable %v\n", b)
			b = b[1:]
		}

	}
	b = b[1:]
	return res, b
}

func (m *MSDPHandler) HandleSB(conn net.Conn, b []byte) {
	//log.Printf("Handling SB string %v\n", b)
	b = b[1:]
	cur := ""

	// Let's try to short circuit with some expected values
	if b[0] == MSDP_VAR {
		b = b[1:]
		var field string
		field, b = readString(b)

		// See if we already know about this
		if val, ok := MSDP_FIELDS[field]; ok {
			log.Printf("Try to intercept %s", field)
			switch val.Type {
			case "string":
				var tmp string
				r := reflect.ValueOf(m)
				f := reflect.Indirect(r).FieldByName(val.Field)
				b = b[1:] // ditch val byte
				tmp, b = readString(b)
				f.SetString(tmp)
				log.Printf("Intercepted value %s, got %s", field, tmp)
				if len(b) == 1 && b[0] == 255 {
					return
				} else {
					log.Printf("Remainder of MSDP interception - %d bytes (%v)", len(b), b)
					return
				}
			case "integer":
				var tmp int
				r := reflect.ValueOf(m)
				f := reflect.Indirect(r).FieldByName(val.Field)
				b = b[1:] // ditch val byte
				tmp, b = readInteger(b)
				f.SetInt(int64(tmp))
				log.Printf("Intercepted value %s, got %d", field, tmp)
				if len(b) == 1 && b[0] == 255 {
					return
				} else {
					log.Printf("Remainder of MSDP interception - %d bytes (%v)", len(b), b)
					return
				}
			case "list":
				var tmp []string
				r := reflect.ValueOf(m)
				f := reflect.Indirect(r).FieldByName(val.Field)

				b = b[1:] // ditch val byte
				tmp, b = readArray(b)
				n := len(tmp)
				if n > f.Cap() {
					ncap := 2 * n
					if ncap < 4 {
						ncap = 4
					}
					nval := reflect.MakeSlice(f.Type(), n, ncap)
					reflect.Copy(nval, f)
					f.Set(nval)
				}
				f.SetLen(n + 1)
				for i := range tmp {
					f.Index(i).SetString(tmp[i])
				}
				log.Printf("Intercepted value %s, got %+v", field, tmp)

				// Special case for REPORTABLE_VARIABLES
				if field == "REPORTABLE_VARIABLES" {
					msg := MSDPMessage([]byte{IAC, SB, m.OptionCode()}, []byte{MSDP_VAR}, []byte("REPORT"))
					for _, msdpvar := range m.Reportables {
						msg = MSDPMessage(msg, []byte{MSDP_VAL}, []byte(msdpvar))
					}
					msg = MSDPMessage(msg, []byte{IAC, SE})
					log.Printf("Got reportables, sending request for all variables")
					conn.Write(msg)
				}

			default:
				log.Printf("Interception of %s failed due to unknown var type '%s'", field, val.Type)
			}
		} else {
			log.Printf("Unknown MSDP field %s", field)
		}

	}

	is_array := false
	array_index := 0
	is_table := false
	for len(b) > 0 {
		switch b[0] {
		case MSDP_VAR:
			b = b[1:]
			cur, b = readString(b)
		case MSDP_VAL:
			b = b[1:]
			if is_array {
				log.Printf("Handling array in SB\n")
			}
			if is_table {
				log.Printf("Handling table in SB\n")
			}
			switch cur {
			case "ACCOUNT_NAME":
				m.AccountName, b = readString(b)
			case "AREA_MAXLEVEL":
				m.AreaMaxLevel, b = readInteger(b)
			case "AREA_MINLEVEL":
				m.AreaMinLevel, b = readInteger(b)
			case "AREA_NAME":
				m.AreaName, b = readString(b)
			case "BANK_GOLD":
				m.BankGold, b = readInteger(b)
			case "COMMANDS":
			case "EXPERIENCE":
				m.Experience, b = readInteger(b)
			case "GOLD":
				m.Gold, b = readInteger(b)
			case "GROUPLEVEL":
				m.GroupLevel, b = readInteger(b)
			case "PC_IN_ROOM":
				var tmp string
				tmp, b = readString(b)
				if tmp == "1" {
					m.PCInRoom = true
				} else {
					m.PCInRoom = false
				}
			case "PC_IN_ZONE":
				var tmp string
				tmp, b = readString(b)
				if tmp == "1" {
					m.PCInZone = true
				} else {
					m.PCInZone = false
				}
			case "PROMPT":
				m.Prompt, b = readString(b)

			case "ROOM_WEATHER":
				m.RoomWeather, b = readString(b)
			case "ROOM_NAME":
				m.RoomName, b = readString(b)
			case "ROOM_TERRAIN":
				m.RoomTerrain, b = readString(b)
			case "ROOM_VNUM":
				m.RoomVnum, b = readString(b)

			case "REPORTABLE_VARIABLES":

			case "SERVER_ID":
				m.Server_ID, b = readString(b)
			case "SHIELD":
				m.Shield, b = readString(b)
			case "STR":
				m.Str, b = readInteger(b)
			case "STR_MAX":
				m.StrMax, b = readInteger(b)
			case "STR_PERM":
				m.StrPerm, b = readInteger(b)
			case "WHOFLAGS":
				m.WhoFlags, b = readString(b)
			case "WIS":
				m.Wis, b = readInteger(b)
			case "WIS_MAX":
				m.WisMax, b = readInteger(b)
			case "WIS_PERM":
				m.WisPerm, b = readInteger(b)
			case "WORLD_TIME":
				m.WorldTime, b = readInteger(b)

			default:
				log.Printf("Unhandled SB VAL on %v: %v\n", cur, b)
			}
			if len(b) == 1 && b[0] == 255 {
				b = b[1:]
			}
		case MSDP_ARRAY_OPEN:
			b = b[1:]
			is_array = true
			log.Printf("Found array %v\n", cur)
			switch cur {
			case "COMMANDS":
				m.Commands, b = readArray(b)
				log.Printf("Got some commands: %+v", m.Commands)
			case "REPORTABLE_VARIABLES":
				m.Reportables, b = readArray(b)
				log.Printf("Got some reportables: %+v", m.Commands)
				msg := MSDPMessage([]byte{IAC, SB, m.OptionCode()}, []byte{MSDP_VAR}, []byte("REPORT"))
				for _, msdpvar := range m.Reportables {
					msg = MSDPMessage(msg, []byte{MSDP_VAL}, []byte(msdpvar))
				}
				msg = MSDPMessage(msg, []byte{IAC, SE})
				conn.Write(msg)

			default:
				log.Printf("Unhandled SB ARRAY OPEN on %v\n", cur)
			}
		case MSDP_ARRAY_CLOSE:
			// Close out array
			log.Printf("Closing array %s\n", cur)
			switch cur {
			case "GROUP":
				for i := array_index; i < 9; i++ {
					m.Group[i] = GroupMember{}

				}
			}
			array_index = 0
			b = b[1:]

		case MSDP_TABLE_OPEN:
			b = b[1:]
			is_table = true
			var x map[string]interface{}
			x, b = readTable(b)
			switch cur {
			case "GROUP":
				//log.Printf("Group entry %d: %v\n", array_index, x)
				x_health, _ := strconv.Atoi(x["health"].(string))
				x_leader, _ := x["is_leader"].(string)
				x_subleader, _ := x["is_subleader"].(string)
				x_level, _ := strconv.Atoi(x["level"].(string))
				x_mana, _ := strconv.Atoi(x["mana"].(string))
				x_stamina, _ := strconv.Atoi(x["stamina"].(string))
				x_race, _ := x["race"].(string)
				x_npc, _ := x["npc"].(string)
				x_with_lead, _ := x["with_leader"].(string)
				x_with_you, _ := x["with_you"].(string)

				m.Group[array_index] = GroupMember{
					Class:    x["class"].(string),
					Flags:    x["flags"].(string),
					Health:   x_health,
					Level:    x_level,
					Name:     x["name"].(string),
					Position: x["position"].(string),
					Stamina:  x_stamina,
					Mana:     x_mana,
					Race:     x_race,
				}
				if x_leader != "0" {
					m.Group[array_index].Is_Leader = true
				}
				if x_subleader != "0" {
					m.Group[array_index].Is_Subleader = true
				}
				if x_npc != "pc" {
					m.Group[array_index].NPC = true
				}
				if x_with_lead != "0" {
					m.Group[array_index].With_Leader = true
				}
				if x_with_you != "0" {
					m.Group[array_index].With_You = true
				}

				array_index += 1

			default:
				log.Printf("Found table %v (parsed %v)\n", cur, x)
			}

		case MSDP_TABLE_CLOSE:
			b = b[1:]
		default:
			if b[0] == 255 && len(b) == 1 {
				b = b[1:]
			} else {
				log.Printf("Unexpected byte: %v\n", b[0])
				b = b[1:]
			}
		}
	}
	is_array = false
	is_table = false
	if len(b) > 0 {
		log.Printf("Remaining b: %v\n", b)
	}
	//fmt.Printf("Found %v\n", cur)
}
