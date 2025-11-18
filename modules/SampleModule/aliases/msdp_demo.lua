-- MSDP Demo Alias: Demonstrates how to access MSDP values from Lua
-- Usage: Type "msdpdemo" or "msdp" to run the demo

session.register_alias("msdp_demo", "^(msdpdemo|msdp)$", function(matches)
    session.output("=== MSDP Values Demo ===\n\n")
    
    -- Example 1: Get a string value (e.g., ROOM_NAME)
    session.output("1. String Values:\n")
    local roomName = session.msdp_get_string("ROOM_NAME")
    if type(roomName) == "string" and roomName ~= "" then
        session.output("   ROOM_NAME: " .. roomName .. "\n")
    else
        session.output("   ROOM_NAME: (not available)\n")
    end
    
    local charName = session.msdp_get_string("CHARACTER_NAME")
    if type(charName) == "string" and charName ~= "" then
        session.output("   CHARACTER_NAME: " .. charName .. "\n")
    else
        session.output("   CHARACTER_NAME: (not available)\n")
    end
    
    -- Example 2: Get an integer value
    session.output("\n2. Integer Values:\n")
    local health = session.msdp_get_int("HEALTH")
    local maxHealth = session.msdp_get_int("HEALTH_MAX")
    if health > 0 or maxHealth > 0 then
        session.output("   HEALTH: " .. tostring(health) .. "\n")
        session.output("   HEALTH_MAX: " .. tostring(maxHealth) .. "\n")
        if maxHealth > 0 then
            local percent = math.floor((health / maxHealth) * 100)
            session.output("   Health Percentage: " .. tostring(percent) .. "%\n")
        end
    else
        session.output("   HEALTH values: (not available)\n")
    end
    
    -- Example 3: Get a boolean value
    session.output("\n3. Boolean Values:\n")
    local inCombat = session.msdp_get_bool("IN_COMBAT")
    session.output("   IN_COMBAT: " .. tostring(inCombat) .. "\n")
    
    -- Example 4: Get an array value
    session.output("\n4. Array Values:\n")
    local exits = session.msdp_get_array("EXITS")
    if exits then
        session.output("   EXITS: [")
        local exitList = {}
        for i = 1, #exits do
            table.insert(exitList, tostring(exits[i]))
        end
        session.output(table.concat(exitList, ", ") .. "]\n")
    else
        session.output("   EXITS: (not available)\n")
    end
    
    -- Example 5: Get a table value
    session.output("\n5. Table Values:\n")
    local roomInfo = session.msdp_get_table("ROOM")
    if roomInfo then
        session.output("   ROOM table:\n")
        for key, value in pairs(roomInfo) do
            if type(value) == "table" then
                session.output("     " .. key .. ": (table)\n")
            else
                session.output("     " .. key .. ": " .. tostring(value) .. "\n")
            end
        end
    else
        session.output("   ROOM table: (not available)\n")
    end
    
    -- Example 6: Get all MSDP data
    session.output("\n6. All MSDP Data:\n")
    local allMsdp = session.msdp_get_all()
    if allMsdp then
        local count = 0
        for key, value in pairs(allMsdp) do
            count = count + 1
        end
        session.output("   Total MSDP variables: " .. tostring(count) .. "\n")
        session.output("   Available keys: ")
        local keys = {}
        for key, _ in pairs(allMsdp) do
            table.insert(keys, key)
        end
        table.sort(keys)
        session.output(table.concat(keys, ", ") .. "\n")
    else
        session.output("   No MSDP data available\n")
    end
    
    session.output("\n=== End of MSDP Demo ===\n")
    session.output("Note: Some values may not be available if not connected or MSDP not negotiated.\n")
end)

