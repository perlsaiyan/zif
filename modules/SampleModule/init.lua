-- SampleModule: A simple example module demonstrating the zif Lua API

-- This module demonstrates:
-- - Module structure with init.lua
-- - Registering aliases from the aliases/ subdirectory

-- The module name and path are available via:
-- module.get_name() and module.get_path()

local module_name = module.get_name()
session.output("SampleModule loaded: " .. module_name .. "\n")

