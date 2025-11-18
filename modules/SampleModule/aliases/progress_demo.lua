-- Progress Bar Demo Alias: Demonstrates progress bar functionality in sidebars
-- Usage: Type "progressdemo" or "pdemo" to run the demo

session.register_alias("progress_demo", "^(progressdemo|pdemo)$", function(matches)
    session.output("=== Progress Bar Demo ===\n")
    
    -- Step 0: Clean up any existing sidebar panes from previous runs
    -- Try to remove sidebar_1 through sidebar_10 (in case of multiple runs)
    session.output("Step 0: Cleaning up any existing sidebar panes...\n")
    for i = 1, 10 do
        local sidebar_id = "sidebar_" .. i
        -- Try to unsplit - it will fail silently if pane doesn't exist
        pcall(function() session.layout_unsplit(sidebar_id) end)
    end
    
    -- Step 1: Create a new sidebar pane and get its ID
    session.add_one_shot_timer("progress_demo_step1", 300, function()
        session.output("\nStep 1: Creating sidebar pane...\n")
        local sidebar_id = session.layout_split("h", "main", "sidebar", 30)
        session.output("Created pane: " .. sidebar_id .. "\n")
        
        -- Wait for the split to complete, then create progress bar
        session.add_one_shot_timer("progress_demo_step2", 500, function()
            session.output("\nStep 2: Creating progress bar in " .. sidebar_id .. "...\n")
            session.progress_create(sidebar_id, 30)
            
            -- Store sidebar_id in closure for later use
            local function update_progress(percent)
                pcall(function() session.progress_update(sidebar_id, percent) end)
            end
            
            local function destroy_progress()
                pcall(function() session.progress_destroy(sidebar_id) end)
            end
            
            local function remove_sidebar()
                pcall(function() session.layout_unsplit(sidebar_id) end)
            end
            
            -- Step 3: Update progress bar incrementally
            local progress = 0.0
            local increment = 0.1
            local update_count = 0
            
            session.add_timer("progress_demo_updater", 200, function()
                update_count = update_count + 1
                progress = progress + increment
                
                if progress > 1.0 then
                    progress = 1.0
                end
                
                update_progress(progress)
                session.output(string.format("Progress: %.0f%%\n", progress * 100))
                
                -- After 10 updates (2 seconds at 200ms intervals), destroy the progress bar
                if update_count >= 10 then
                    session.remove_timer("progress_demo_updater")
                    
                    -- Wait a moment to show the completed bar, then destroy it
                    session.add_one_shot_timer("progress_demo_step4", 1000, function()
                        session.output("\nStep 4: Destroying progress bar...\n")
                        destroy_progress()
                        session.output("Progress bar destroyed!\n")
                        
                        -- Step 5: Remove the sidebar pane
                        session.add_one_shot_timer("progress_demo_step5", 500, function()
                            session.output("\nStep 5: Removing sidebar pane...\n")
                            remove_sidebar()
                            session.output("=== End of Progress Bar Demo ===\n")
                        end)
                    end)
                end
            end)
        end)
    end)
end)

