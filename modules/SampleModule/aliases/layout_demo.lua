-- Layout Demo Alias: Showcases the layout system capabilities
-- Usage: Type "layoutdemo" or "ldemo" to run the demo

session.register_alias("layout_demo", "^(layoutdemo|ldemo)$", function(matches)
    session.output("=== Layout System Demo ===\n")
    
    -- Step 1: Split main pane horizontally to create a sidebar (30% left, 70% right)
    session.output("Step 1: Splitting main pane horizontally (30% sidebar, 70% main)...\n")
    session.layout_split("h", "main", "sidebar", 30)
    
    -- Wait a moment for the split to complete, then set content on sidebar
    session.add_one_shot_timer("layout_demo_step2", 500, function()
        session.output("\nStep 2: Setting content on sidebar pane...\n")
        -- The sidebar pane ID is auto-generated (e.g., sidebar_1, sidebar_2, etc.)
        -- Try common IDs, starting with sidebar_1
        local sidebar_content = 
            "+-------------------------------+\n" ..
            "|   Layout Demo Sidebar         |\n" ..
            "+-------------------------------+\n" ..
            "| This is a sidebar pane!       |\n" ..
            "|                               |\n" ..
            "| You can see this content      |\n" ..
            "| because we set it using:      |\n" ..
            "|                               |\n" ..
            "| session.layout_set_content()  |\n" ..
            "|                               |\n" ..
            "| The pane was created by       |\n" ..
            "| splitting the main pane       |\n" ..
            "| horizontally.                 |\n" ..
            "|                               |\n" ..
            "| Try scrolling this pane!      |\n" ..
            "|                               |\n" ..
            "| Pane ID: sidebar_1            |\n" ..
            "+-------------------------------+\n"
        
        -- Try setting content on sidebar_1 (most common first ID)
        session.layout_set_content("sidebar_1", sidebar_content)
        
        -- Step 3: List all panes
        session.add_one_shot_timer("layout_demo_step3", 500, function()
            session.output("\nStep 3: Listing all panes...\n")
            session.layout_list_panes()
            
            -- Step 4: Get info about the main pane
            session.add_one_shot_timer("layout_demo_step4", 500, function()
                session.output("\nStep 4: Getting info about 'main' pane...\n")
                session.layout_pane_info("main")
                
                -- Step 5: Focus on the sidebar
                session.add_one_shot_timer("layout_demo_step5", 500, function()
                    session.output("\nStep 5: Focusing sidebar pane...\n")
                    session.output("(Note: Sidebar pane ID is auto-generated, e.g., 'sidebar_1')\n")
                    session.output("You can use #focus <pane_id> to focus different panes.\n")
                    
                    -- Step 6: Demo complete
                    session.add_one_shot_timer("layout_demo_step6", 1000, function()
                        session.output("\nStep 6: Demo complete!\n")
                        session.output("To remove the sidebar, use: session.layout_unsplit('sidebar_1')\n")
                        session.output("Or use the command: #unsplit sidebar_1\n")
                        session.output("=== End of Layout Demo ===\n")
                    end)
                end)
            end)
        end)
    end)
end)

