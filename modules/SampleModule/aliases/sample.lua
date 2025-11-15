-- Sample alias: When user types "sample", send "dance" to MUD and output a message

session.register_alias("sample", "^sample$", function(matches)
    -- matches[1] is the full match, matches[2] would be first capture group, etc.
    session.output("Making me dance!\n")
    session.send("dance")
end)

