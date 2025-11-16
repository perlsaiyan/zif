-- Example trigger: Outputs a message when "TRIGGER" appears in any line

session.register_trigger("trigger_example", "TRIGGER", function(ansi_line, line, matches)
    -- ansi_line: The full line with ANSI color codes
    -- line: The line with ANSI codes stripped
    -- matches: Array of regex capture groups (matches[1] is the full match)
    session.output("I SAW 'TRIGGER'\n")
end, false)  -- false = don't match on color codes, match on stripped text

