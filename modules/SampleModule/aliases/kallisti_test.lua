-- Test kallisti context injection
-- This alias demonstrates that Lua modules can access plugin-provided functions and values

session.register_alias("kallisti_test", "^kallisti_test$", function(matches)
    -- Test function_test()
    if kallisti and kallisti.function_test then
        local result = kallisti.function_test()
        session.output("kallisti.function_test() returned: " .. result .. "\n")
    else
        session.output("kallisti.function_test() not available (plugin may not be loaded)\n")
    end
    
    -- Test last_line value
    if kallisti and kallisti.last_line then
        local lastLine = kallisti.last_line
        session.output("kallisti.last_line = " .. tostring(lastLine) .. "\n")
        if lastLine > 0 then
            -- Convert nanoseconds to seconds for display
            local seconds = lastLine / 1000000000
            session.output("  (Last MUD line received " .. string.format("%.2f", seconds) .. " seconds ago)\n")
        else
            session.output("  (No MUD lines received yet)\n")
        end
    else
        session.output("kallisti.last_line not available (plugin may not be loaded)\n")
    end
end)


session.register_alias("kallisti_room", "^kallisti_room$", function(matches)
    if kallisti and kallisti.current_room then
        local room = kallisti.current_room()
        if room then
            session.output("Room Information:\n")
            session.output("  Title: " .. (room.title or "Unknown") .. "\n")
            session.output("  Vnum: " .. (room.vnum or "Unknown") .. "\n")
            
            if room.exits then
                session.output("  Exits: " .. #room.exits .. "\n")
            end
            
            if room.objects then
                session.output("  Objects: " .. #room.objects .. "\n")
                for i, obj in ipairs(room.objects) do
                    session.output("    - " .. obj .. "\n")
                end
            end
            
            if room.mobs then
                session.output("  Mobs: " .. #room.mobs .. "\n")
                for i, mob in ipairs(room.mobs) do
                    session.output("    - " .. mob .. "\n")
                end
            end
        else
            session.output("No room information available.\n")
        end
    else
        session.output("kallisti.current_room not available.\n")
    end
end)
