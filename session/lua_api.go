package session

import (
	"fmt"
	"log"
	"regexp"
	"runtime/debug"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// LuaModuleContextKey is the key used to store the current module context in Lua state
const LuaModuleContextKey = "__zif_current_module"

// SetCurrentModule sets the current module context in the Lua state
func SetCurrentModule(L *lua.LState, moduleName string) {
	L.SetField(L.Get(lua.RegistryIndex), LuaModuleContextKey, lua.LString(moduleName))
}

// GetCurrentModule gets the current module context from the Lua state
func GetCurrentModule(L *lua.LState) string {
	module := L.GetField(L.Get(lua.RegistryIndex), LuaModuleContextKey)
	if module.Type() == lua.LTString {
		return string(module.(lua.LString))
	}
	return ""
}

// RegisterLuaAPI registers all the Lua API functions with the session's Lua state
func (s *Session) RegisterLuaAPI() {
	L := s.LuaState

	// Create session table
	sessionMT := L.NewTypeMetatable("session")
	L.SetGlobal("session", sessionMT)

	// session:send(command)
	L.SetField(sessionMT, "send", L.NewFunction(func(L *lua.LState) int {
		command := L.CheckString(1)
		if s.Connected && s.Socket != nil {
			s.Socket.Write([]byte(command + LineTerminator))
		}
		return 0
	}))

	// session:output(text)
	L.SetField(sessionMT, "output", L.NewFunction(func(L *lua.LState) int {
		text := L.CheckString(1)
		s.Output(text)
		return 0
	}))

	// session:get_data(key)
	L.SetField(sessionMT, "get_data", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if val, ok := s.Data[key]; ok {
			L.Push(luaToLValue(L, val))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))

	// session:set_data(key, value)
	L.SetField(sessionMT, "set_data", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		value := L.CheckAny(2)
		s.Data[key] = lValueToGo(value)
		return 0
	}))

	// session:register_trigger(name, pattern, func, color)
	L.SetField(sessionMT, "register_trigger", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		pattern := L.CheckString(2)
		fn := L.CheckFunction(3)
		color := false
		if L.GetTop() >= 4 {
			color = L.ToBool(4)
		}

		moduleName := GetCurrentModule(L)
		if moduleName == "" {
			L.RaiseError("register_trigger called outside of module context")
			return 0
		}

		// Compile regex
		re, err := regexp.Compile(pattern)
		if err != nil {
			L.RaiseError("invalid regex pattern: %v", err)
			return 0
		}

		// Create action/trigger
		action := Action{
			Name:    name,
			Pattern: pattern,
			Color:   color,
			Enabled: true,
			RE:      re,
			Fn: func(sess *Session, matches ActionMatches) {
				defer func() {
					if r := recover(); r != nil {
						stack := debug.Stack()
						logPanic(fmt.Sprintf("Lua trigger %s", name), r, stack)
						if sess != nil {
							sess.Output(fmt.Sprintf("\nPANIC in Lua trigger %s: %v\n(Check ~/.config/zif/panic.log for details)\n", name, r))
						}
					}
				}()
				// Call Lua function with matches
				L := sess.LuaState
				L.Push(fn)
				L.Push(lua.LString(matches.ANSILine))
				L.Push(lua.LString(matches.Line))
				
				// Push matches array
				matchesTable := L.NewTable()
				for i, match := range matches.Matches {
					L.RawSetInt(matchesTable, i+1, lua.LString(match))
				}
				L.Push(matchesTable)
				
				if err := L.PCall(3, 0, nil); err != nil {
					log.Printf("Error calling Lua trigger %s: %v", name, err)
				}
			},
			Count: 0,
		}

		s.AddAction(action)

		// Track in module registry
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				module.Triggers = append(module.Triggers, name)
			}
		}

		return 0
	}))

	// session:register_alias(name, pattern, func)
	L.SetField(sessionMT, "register_alias", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		pattern := L.CheckString(2)
		fn := L.CheckFunction(3)

		moduleName := GetCurrentModule(L)
		if moduleName == "" {
			L.RaiseError("register_alias called outside of module context")
			return 0
		}

		// Compile regex
		re, err := regexp.Compile(pattern)
		if err != nil {
			L.RaiseError("invalid regex pattern: %v", err)
			return 0
		}

		// Create alias
		alias := Alias{
			Name:    name,
			Pattern: pattern,
			RE:      re,
			Fn: func(sess *Session, matches []string) {
				defer func() {
					if r := recover(); r != nil {
						stack := debug.Stack()
						logPanic(fmt.Sprintf("Lua alias %s", name), r, stack)
						if sess != nil {
							sess.Output(fmt.Sprintf("\nPANIC in Lua alias %s: %v\n(Check ~/.config/zif/panic.log for details)\n", name, r))
						}
					}
				}()
				// Call Lua function with matches
				L := sess.LuaState
				L.Push(fn)
				
				// Push matches array
				matchesTable := L.NewTable()
				for i, match := range matches {
					L.RawSetInt(matchesTable, i+1, lua.LString(match))
				}
				L.Push(matchesTable)
				
				if err := L.PCall(1, 0, nil); err != nil {
					log.Printf("Error calling Lua alias %s: %v", name, err)
				}
			},
			Enabled: true,
		}

		s.AddAlias(alias)

		// Track in module registry
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				module.Aliases = append(module.Aliases, name)
			}
		}

		return 0
	}))

	// session:add_timer(name, interval_ms, func)
	L.SetField(sessionMT, "add_timer", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		interval := L.CheckInt(2)
		fn := L.CheckFunction(3)

		moduleName := GetCurrentModule(L)
		if moduleName == "" {
			L.RaiseError("add_timer called outside of module context")
			return 0
		}

		// Create timer
		ticker := &TickerRecord{
			Name:     name,
			Interval: interval,
			Fn: func(sess *Session) {
				defer func() {
					if r := recover(); r != nil {
						stack := debug.Stack()
						logPanic(fmt.Sprintf("Lua timer %s", name), r, stack)
						if sess != nil {
							sess.Output(fmt.Sprintf("\nPANIC in Lua timer %s: %v\n(Check ~/.config/zif/panic.log for details)\n", name, r))
						}
					}
				}()
				// Call Lua function
				L := sess.LuaState
				L.Push(fn)
				if err := L.PCall(0, 0, nil); err != nil {
					log.Printf("Error calling Lua timer %s: %v", name, err)
				}
			},
			NextFire: s.Birth.Add(0), // Will be set properly by AddLuaTimer
			LastFire: s.Birth,
			Count:    0,
		}

		s.AddLuaTimer(ticker)

		// Track in module registry
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				module.Timers = append(module.Timers, name)
			}
		}

		return 0
	}))

	// session:remove_timer(name)
	L.SetField(sessionMT, "remove_timer", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		moduleName := GetCurrentModule(L)
		if moduleName == "" {
			L.RaiseError("remove_timer called outside of module context")
			return 0
		}

		// Remove timer from registry
		s.RemoveLuaTimer(name)

		// Remove from module tracking
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				for i, timerName := range module.Timers {
					if timerName == name {
						// Remove from slice
						module.Timers = append(module.Timers[:i], module.Timers[i+1:]...)
						break
					}
				}
			}
		}

		return 0
	}))

	// session:add_one_shot_timer(name, delay_ms, func)
	L.SetField(sessionMT, "add_one_shot_timer", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		delay := L.CheckInt(2)
		fn := L.CheckFunction(3)

		moduleName := GetCurrentModule(L)
		if moduleName == "" {
			L.RaiseError("add_one_shot_timer called outside of module context")
			return 0
		}

		// Create one-shot timer
		ticker := &TickerRecord{
			Name:     name,
			Interval: delay, // Not used for one-shot, but set for consistency
			Fn: func(sess *Session) {
				defer func() {
					if r := recover(); r != nil {
						stack := debug.Stack()
						logPanic(fmt.Sprintf("Lua one-shot timer %s", name), r, stack)
						if sess != nil {
							sess.Output(fmt.Sprintf("\nPANIC in Lua one-shot timer %s: %v\n(Check ~/.config/zif/panic.log for details)\n", name, r))
						}
					}
				}()
				// Call Lua function
				L := sess.LuaState
				L.Push(fn)
				if err := L.PCall(0, 0, nil); err != nil {
					log.Printf("Error calling Lua one-shot timer %s: %v", name, err)
				}
				// Remove timer after firing
				sess.RemoveLuaTimer(name)
				// Remove from module tracking
				if sess.Modules != nil {
					if module, ok := sess.Modules.Modules[moduleName]; ok {
						for i, timerName := range module.Timers {
							if timerName == name {
								module.Timers = append(module.Timers[:i], module.Timers[i+1:]...)
								break
							}
						}
					}
				}
			},
			NextFire: time.Now().Add(time.Duration(delay) * time.Millisecond),
			LastFire: s.Birth,
			Count:    0,
		}

		s.AddLuaTimer(ticker)

		// Track in module registry
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				module.Timers = append(module.Timers, name)
			}
		}

		return 0
	}))

	// session:get_ringlog(limit)
	L.SetField(sessionMT, "get_ringlog", L.NewFunction(func(L *lua.LState) int {
		_ = 100 // limit - reserved for future implementation
		if L.GetTop() >= 1 {
			_ = L.CheckInt(1)
		}

		// Get ringlog entries (simplified - would need to implement proper ringlog query)
		result := L.NewTable()
		// TODO: Implement proper ringlog query
		L.Push(result)
		return 1
	}))

	// Create module table
	moduleMT := L.NewTypeMetatable("module")
	L.SetGlobal("module", moduleMT)

	// module:get_name()
	L.SetField(moduleMT, "get_name", L.NewFunction(func(L *lua.LState) int {
		moduleName := GetCurrentModule(L)
		L.Push(lua.LString(moduleName))
		return 1
	}))

	// module:get_path()
	L.SetField(moduleMT, "get_path", L.NewFunction(func(L *lua.LState) int {
		moduleName := GetCurrentModule(L)
		if s.Modules != nil {
			if module, ok := s.Modules.Modules[moduleName]; ok {
				L.Push(lua.LString(module.Path))
				return 1
			}
		}
		L.Push(lua.LString(""))
		return 1
	}))
}

// Helper functions to convert between Lua values and Go values

func luaToLValue(L *lua.LState, val interface{}) lua.LValue {
	switch v := val.(type) {
	case string:
		return lua.LString(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case bool:
		return lua.LBool(v)
	default:
		return lua.LNil
	}
}

func lValueToGo(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case lua.LBool:
		return bool(v)
	default:
		return nil
	}
}

