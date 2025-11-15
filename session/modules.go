package session

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/perlsaiyan/zif/config"
	lua "github.com/yuin/gopher-lua"
)

// Module represents a loaded Lua module
type Module struct {
	Name     string
	Path     string
	Enabled  bool
	Triggers []string
	Aliases  []string
	Timers   []string
}

// ModuleRegistry tracks all loaded modules for a session
type ModuleRegistry struct {
	Modules map[string]*Module
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		Modules: make(map[string]*Module),
	}
}

// LoadGlobalModules loads all modules from the global modules directory
func LoadGlobalModules(s *Session) error {
	globalModulesDir, err := config.GetGlobalModulesDir()
	if err != nil {
		return fmt.Errorf("failed to get global modules directory: %v", err)
	}

	return loadModulesFromDir(s, globalModulesDir)
}

// LoadSessionModules loads all modules from a session's modules directory
func LoadSessionModules(s *Session, sessionName string) error {
	sessionModulesDir, err := config.GetSessionModulesDir(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session modules directory: %v", err)
	}

	return loadModulesFromDir(s, sessionModulesDir)
}

// loadModulesFromDir discovers and loads modules from a directory
func loadModulesFromDir(s *Session, dir string) error {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory doesn't exist, that's okay
		return nil
	}

	// Read directory entries
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read modules directory %s: %v", dir, err)
	}

	// Find all directories that contain init.lua
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modulePath := filepath.Join(dir, entry.Name())
		initPath := filepath.Join(modulePath, "init.lua")

		// Check if init.lua exists
		if _, err := os.Stat(initPath); os.IsNotExist(err) {
			continue
		}

		// Load the module
		if err := LoadModule(s, modulePath); err != nil {
			log.Printf("Failed to load module %s: %v", entry.Name(), err)
			continue
		}
	}

	return nil
}

// LoadModule loads a single module from a directory path
func LoadModule(s *Session, modulePath string) error {
	moduleName := filepath.Base(modulePath)
	initPath := filepath.Join(modulePath, "init.lua")

	// Check if init.lua exists
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return fmt.Errorf("init.lua not found in %s", modulePath)
	}

	// Create module entry
	module := &Module{
		Name:     moduleName,
		Path:     modulePath,
		Enabled:  true,
		Triggers: make([]string, 0),
		Aliases:  make([]string, 0),
		Timers:   make([]string, 0),
	}

	// Register module before loading (so it can track registrations)
	s.Modules.Modules[moduleName] = module

	// Set current module context
	SetCurrentModule(s.LuaState, moduleName)

	// Register Lua API if not already done
	if s.LuaState.GetGlobal("session").Type() == lua.LTNil {
		s.RegisterLuaAPI()
	}

	// Read and execute init.lua
	initContent, err := ioutil.ReadFile(initPath)
	if err != nil {
		return fmt.Errorf("failed to read init.lua: %v", err)
	}

	// Execute init.lua
	if err := s.LuaState.DoString(string(initContent)); err != nil {
		return fmt.Errorf("failed to execute init.lua: %v", err)
	}

	// Load triggers, aliases, and scripts from subdirectories
	if err := loadModuleSubdirs(s, modulePath, moduleName); err != nil {
		log.Printf("Warning: failed to load subdirectories for module %s: %v", moduleName, err)
	}

	log.Printf("Loaded module: %s from %s", moduleName, modulePath)
	return nil
}

// loadModuleSubdirs loads triggers/, aliases/, and scripts/ subdirectories
func loadModuleSubdirs(s *Session, modulePath string, moduleName string) error {
	// Load triggers
	triggersDir := filepath.Join(modulePath, "triggers")
	if entries, err := ioutil.ReadDir(triggersDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".lua") {
				triggerPath := filepath.Join(triggersDir, entry.Name())
				content, err := ioutil.ReadFile(triggerPath)
				if err != nil {
					log.Printf("Failed to read trigger %s: %v", triggerPath, err)
					continue
				}
				if err := s.LuaState.DoString(string(content)); err != nil {
					log.Printf("Failed to execute trigger %s: %v", triggerPath, err)
				}
			}
		}
	}

	// Load aliases
	aliasesDir := filepath.Join(modulePath, "aliases")
	if entries, err := ioutil.ReadDir(aliasesDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".lua") {
				aliasPath := filepath.Join(aliasesDir, entry.Name())
				content, err := ioutil.ReadFile(aliasPath)
				if err != nil {
					log.Printf("Failed to read alias %s: %v", aliasPath, err)
					continue
				}
				if err := s.LuaState.DoString(string(content)); err != nil {
					log.Printf("Failed to execute alias %s: %v", aliasPath, err)
				}
			}
		}
	}

	// Load scripts
	scriptsDir := filepath.Join(modulePath, "scripts")
	if entries, err := ioutil.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".lua") {
				scriptPath := filepath.Join(scriptsDir, entry.Name())
				content, err := ioutil.ReadFile(scriptPath)
				if err != nil {
					log.Printf("Failed to read script %s: %v", scriptPath, err)
					continue
				}
				if err := s.LuaState.DoString(string(content)); err != nil {
					log.Printf("Failed to execute script %s: %v", scriptPath, err)
				}
			}
		}
	}

	return nil
}

// EnableModule enables a module and all its triggers/aliases/timers
func (s *Session) EnableModule(moduleName string) error {
	module, ok := s.Modules.Modules[moduleName]
	if !ok {
		return fmt.Errorf("module %s not found", moduleName)
	}

	if module.Enabled {
		return nil // Already enabled
	}

	module.Enabled = true

	// Enable all triggers
	for _, triggerName := range module.Triggers {
		if action, ok := s.Actions.Actions[triggerName]; ok {
			action.Enabled = true
			s.Actions.Actions[triggerName] = action
		}
	}

	// Enable all aliases
	for _, aliasName := range module.Aliases {
		if alias, ok := s.Aliases.Aliases[aliasName]; ok {
			alias.Enabled = true
			s.Aliases.Aliases[aliasName] = alias
		}
	}

	// Timers are automatically enabled when added to TickerRegistry
	log.Printf("Enabled module: %s", moduleName)
	return nil
}

// DisableModule disables a module and all its triggers/aliases/timers
func (s *Session) DisableModule(moduleName string) error {
	module, ok := s.Modules.Modules[moduleName]
	if !ok {
		return fmt.Errorf("module %s not found", moduleName)
	}

	if !module.Enabled {
		return nil // Already disabled
	}

	module.Enabled = false

	// Disable all triggers
	for _, triggerName := range module.Triggers {
		if action, ok := s.Actions.Actions[triggerName]; ok {
			action.Enabled = false
			s.Actions.Actions[triggerName] = action
		}
	}

	// Disable all aliases
	for _, aliasName := range module.Aliases {
		if alias, ok := s.Aliases.Aliases[aliasName]; ok {
			alias.Enabled = false
			s.Aliases.Aliases[aliasName] = alias
		}
	}

	// Remove all timers
	for _, timerName := range module.Timers {
		s.RemoveLuaTimer(timerName)
	}

	log.Printf("Disabled module: %s", moduleName)
	return nil
}

