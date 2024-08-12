package main

import (
	"bufio"
	"fmt"
	"os"
	"html"
	"net"
	"path/filepath"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"log"

)

const (
    configDir = "/.config/abbtr/"
    configFileName = "abbtr.conf"
    logDir = "/.local/share/abbtr/"
    logFileName = "abbtr.log"
    VERSION = "1.0.58"
)

var configFile = filepath.Join(os.Getenv("HOME"), configDir, configFileName)

var reservedNames = []string{
    "-h", "-l", "-n", "-r", "-c", "-ln", "-v", "-i", "-e", "-b",
    "-H", "-L", "-N", "-R", "-C", "-LN", "-V", "-I", "-E", "-B",
    "-lN", "-Ln",

    // Reserved for future implementations
    "-g", "-G", "-w", "-W", "-t", "-T", "-x", "-X", "-y", "-Y",
    "-z", "-Z", "-a", "-A",

    // Reserved to system commands
    "su", "passwd", "clear", "exit", "logout", "reset", "whoami", "hostname",
    "sync", "uptime", "pwd", "yes", "true", "false", "cal", "date", "arch",
    "bg", "fg", "jobs", "tset", "lsblk",
}

func main() {

    // Initialize the config file
    homeDir, err := os.UserHomeDir()
    if err != nil {
        log.Fatalf("Failed to get home directory: %v", err)
    }
    configFile = filepath.Join(homeDir, ".config", "abbtr", "abbtr.conf")

    err = initConfigFile()
    if err != nil {
        log.Fatalf("Failed to initialize config file: %v", err)
    }

    // Verify if ~/.local/bin is in the PATH
    checkPath()

    // Call for syncRulesWithScripts
    err = syncRulesWithScripts()
    if err != nil {
        fmt.Printf("Warning: Unable to synchronize rules with scripts: %v\n", err)
        fmt.Println("This may be normal if this is the first run or if ~/.local/bin doesn't exist.")
        fmt.Println("The program will continue, but some functionality may be limited.")
    }

    args := os.Args[1:]

    bottleValues := make(map[string]string)
    var commands []string

    for i := 0; i < len(args); i++ {
        if strings.HasPrefix(args[i], "-b=") {
            parts := strings.SplitN(args[i], "=", 2)
            if len(parts) == 2 {
                bottleParts := strings.SplitN(parts[1], ":", 2)
                if len(bottleParts) == 2 {
                    bottleValues[bottleParts[0]] = bottleParts[1]
                }
            }
        } else {
            commands = append(commands, args[i])
        }
    }

    if len(commands) == 0 {
        showHelp()
        return
    }

    switch commands[0] {
    case "-h":
        showHelp()
    case "-l":
        listRules()
    case "-n":
        if len(commands) < 3 {
            fmt.Println("Error: Incorrect usage of -n. It should be: abbtr -n <name> '<command>'")
            return
        }
        name := commands[1]
        command := strings.Join(commands[2:], " ")
        createRule(name, command)
    case "-r":
        if len(commands) == 1 {
            fmt.Println("Error: Incorrect usage of -r. It should be: abbtr -r <name> [<name>...] or abbtr -r a")
            return
        }
        names := commands[1:]
        if len(names) == 1 && names[0] == "a" {
            deleteAllRules()
        } else {
            for _, name := range names {
                deleteRule(name)
            }
        }
    case "-c":
        if len(commands) < 3 {
            fmt.Println("Error: Incorrect usage of -c. It should be: abbtr -c <name> '<command>'")
            return
        }
        name := commands[1]
        command := strings.Join(commands[2:], " ")
        updateRule(name, command)
    case "-ln":
        if len(commands) != 2 {
            fmt.Println("Error: Incorrect usage of -ln. It should be: abbtr -ln <name>")
            return
        }
        name := commands[1]
        showRule(name)
    case "-v":
        fmt.Println("abbtr version", VERSION)
    case "-i":
        if len(commands) != 2 {
            fmt.Println("Error: Incorrect usage of -i. It should be: abbtr -i <file path>")
            return
        }
        importSource := commands[1]
        importRulesFromFile(importSource)
    case "-e":
        exportRules()
    default:
        if strings.HasPrefix(commands[0], "-") {
            fmt.Println("Unrecognized option. Use abbtr -h to see the available options.")
        } else {
            runCommands(commands, bottleValues)
        }
    }
}

func showHelp() {
    fmt.Println("Usage: abbtr <option>")
    fmt.Println(" ")
    fmt.Println("Available options:")
    fmt.Println(" -n <name> '<command>'\tCreate a new rule")
    fmt.Println(" -l\t\t\tList stored rules")
    fmt.Println(" -r <name> [<name>...]\tDelete existing rules")
    fmt.Println(" -r a \t\t\tDelete all rules")
    fmt.Println(" -c <name> '<command>'\tUpdate the command of a rule")
    fmt.Println(" -ln <name>\t\tShow the contents of a specific rule")
    fmt.Println(" -h\t\t\tShow this help")
    fmt.Println(" -v\t\t\tShow the program version")
    fmt.Println(" -i <file path>\t\tImport rules from a local file")
    fmt.Println(" -e\t\t\tExport rules to a text file (backup)")
    fmt.Println(" -b=<variable:value>\tPre-define the content of a bottle")
    fmt.Println("\t\t\tSyntax for create bottles: b%('variable')%b")
    fmt.Println(" ")
    fmt.Println("Usage examples:")
    fmt.Println(" Create a new rule: abbtr -n update 'sudo apt update -y'")
    fmt.Println(" The next time just run: update")
    fmt.Println(" ")
    fmt.Println(" Create a new rule with bottle: abbtr -n ssh 'ssh -p 2222 b%('username')%b@example.com'")
    fmt.Println(" The next time you run 'ssh' the system will ask you for the username value")
    fmt.Println(" ")
    fmt.Println("For further help go to https://github.com/manuwarfare/abbtr")
    fmt.Println("Author: Manuel Guerra")
    fmt.Printf("V %s | This software is licensed under the GNU GPLv3\n", VERSION)
}

func listRules() {
    file, err := os.Open(configFile)
    if err != nil {
        fmt.Println("Failed to open the configuration file:", err)
        fmt.Println("No rules have been created in abbtr yet.")
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var rules [][]string

    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        name := strings.TrimSpace(parts[0])
        command := strings.TrimSpace(parts[1])
        rules = append(rules, []string{name, command})
    }

    if len(rules) == 0 {
        fmt.Println("No rules have been created in abbtr yet.")
        return
    }

    // Print rules
    fmt.Println("Rules:")
    for _, rule := range rules {
        fmt.Printf("Rule Name: %s\n", rule[0])
        fmt.Printf("Command: %s\n\n", rule[1])
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading the configuration file:", err)
    }
}

func createRule(name, command string) {
    lines, err := readLines(configFile)
    if err != nil {
        fmt.Println("Error reading the configuration file:", err)
        return
    }

    if isReservedName(name) {
        fmt.Printf("Unable to create a rule with this name. '%s' is a reserved command name.\n", name)
        return
    }

    found := false
    for i, line := range lines {
        if strings.HasPrefix(line, name+" = ") {
            fmt.Printf("The rule '%s' already exists. Do you want to overwrite it? (y/n): ", name)
            var response string
            fmt.Scanln(&response)
            if response != "y" {
                fmt.Println("Operation cancelled.")
                return
            }
            lines[i] = fmt.Sprintf("%s = %s", name, command)
            found = true
            break
        }
    }

    if !found {
        lines = append(lines, fmt.Sprintf("%s = %s", name, command))
    }

    err = writeLines(configFile, lines)
    if err != nil {
        fmt.Println("Error writing to the configuration file:", err)
        return
    }

    // Create the directory ~/.local/bin if it does not exist
    binDir := filepath.Join(os.Getenv("HOME"), ".local/bin")
    err = os.MkdirAll(binDir, 0755)
    if err != nil {
        fmt.Printf("Error creating directory %s: %v\n", binDir, err)
        return
    }

    // Create the script in ~/.local/bin using os.WriteFile
    scriptPath := filepath.Join(os.Getenv("HOME"), ".local/bin", name)
    scriptContent := fmt.Sprintf(`#!/bin/bash
start=$(date +%%s.%%N)
%s
end=$(date +%%s.%%N)
duration=$(echo "$end - $start" | bc)
echo "[$(date +'%%Y-%%m-%%d %%H:%%M:%%S')] EXECUTE_RULE %s at $(hostname -I | awk '{print $1}') | Rule: %s, Command: "%s", Result: Success, Duration: ${duration}s" >> %s
`, command, os.Getenv("USER"), name, command, filepath.Join(os.Getenv("HOME"), logDir, logFileName))

    err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
    if err != nil {
        fmt.Printf("Error creating script: %v\n", err)
        return
    }

    // Log the event in abbtr.log
    err = logEvent("CREATE_RULE", fmt.Sprintf("Name: %s, Command: %s", name, command))
    if err != nil {
        fmt.Printf("Warning: Failed to log event: %v\n", err)
    }

    fmt.Printf("Rule '%s' successfully added. You can now use it directly by typing '%s'\n", name, name)
}

func deleteRule(name string) {
    lines, err := readLines(configFile)
    if err != nil {
        fmt.Println("Error reading the configuration file:", err)
        return
    }

    // Check if the rule exists and remove it from the configuration file
    found := false
    for i, line := range lines {
        if strings.HasPrefix(line, name+" = ") {
            lines = append(lines[:i], lines[i+1:]...)
            found = true
            break
        }
    }

    if !found {
        fmt.Printf("Rule '%s' not found.\n", name)
        return
    }

    err = writeLines(configFile, lines)
    if err != nil {
        fmt.Println("Error writing to the configuration file:", err)
        return
    }

    // Remove the corresponding script in ~/.local/bin
    scriptPath := filepath.Join(os.Getenv("HOME"), ".local/bin", name)
    err = os.Remove(scriptPath)
    if err != nil && !os.IsNotExist(err) {
        fmt.Printf("Error deleting script: %v\n", err)
        return
    }

    // Log the deletion event in abbtr.log
    err = logEvent("DELETE_RULE", fmt.Sprintf("Name: %s", name))
    if err != nil {
        fmt.Printf("Warning: Failed to log event: %v\n", err)
    }

    fmt.Printf("Rule '%s' successfully deleted.\n", name)
}

func deleteAllRules() error {
    // Open the configuration file for writing
    file, err := os.OpenFile(configFile, os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("failed to open abbtr.conf: %v", err)
    }
    defer file.Close()

    // Truncate the configuration file to remove all rules
    err = file.Truncate(0)
    if err != nil {
        return fmt.Errorf("failed to truncate abbtr.conf: %v", err)
    }

    // Define the directory containing the scripts for the rules
    rulesDir := filepath.Join(os.Getenv("HOME"), ".local/bin")

    // Read all files in the rules directory
    files, err := os.ReadDir(rulesDir)
    if err != nil {
        return fmt.Errorf("failed to read rules directory: %v", err)
    }

    // Iterate over the files and remove each script
    for _, file := range files {
        if !file.IsDir() {  // Ensure it's not a directory
            scriptPath := filepath.Join(rulesDir, file.Name())
            err := os.Remove(scriptPath)
            if err != nil {
                fmt.Printf("Error deleting script %s: %v\n", file.Name(), err)
            }
        }
    }

    fmt.Println("All rules have been successfully deleted.")
    return nil
}

func updateRule(name, command string) {
    // Initialize configuration file
    err := initConfigFile()
    if err != nil {
        fmt.Printf("Error initializing config file: %v\n", err)
        return
    }

    // Read existing lines from the configuration file
    lines, err := readLines(configFile)
    if err != nil {
        fmt.Println("Error reading the configuration file:", err)
        return
    }

    // Check if the rule name is reserved
    if isReservedName(name) {
        fmt.Printf("Unable to update rule. '%s' is a reserved command name.\n", name)
        return
    }

    // Update the rule in the configuration
    found := false
    for i, line := range lines {
        if strings.HasPrefix(line, name+" = ") {
            lines[i] = fmt.Sprintf("%s = %s", name, command)
            found = true
            break
        }
    }

    if !found {
        fmt.Printf("Rule '%s' not found.\n", name)
        return
    }

    // Write updated lines to the configuration file
    err = writeLines(configFile, lines)
    if err != nil {
        fmt.Println("Error writing to the configuration file:", err)
        return
    }

    // Create the directory for scripts if it does not exist
    binDir := filepath.Join(os.Getenv("HOME"), ".local/bin")
    err = os.MkdirAll(binDir, 0755)
    if err != nil {
        fmt.Printf("Error creating directory %s: %v\n", binDir, err)
        return
    }

    // Create or update the script file
    scriptPath := filepath.Join(binDir, name)
    scriptContent := fmt.Sprintf(`#!/bin/bash
start=$(date +%%s.%%N)
%s
end=$(date +%%s.%%N)
duration=$(echo "$end - $start" | bc)
echo "[$(date +'%%Y-%%m-%%d %%H:%%M:%%S')] UPDATE_RULE %s at $(hostname -I | awk '{print $1}') | Rule: %s, Command: "%s", Result: Success, Duration: ${duration}s" >> %s
`, command, os.Getenv("USER"), name, command, filepath.Join(os.Getenv("HOME"), logDir, logFileName))

    err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
    if err != nil {
        fmt.Printf("Error updating script: %v\n", err)
        return
    }

    // Log the event
    err = logEvent("UPDATE_RULE", fmt.Sprintf("Name: %s, New Command: %s", name, command))
    if err != nil {
        fmt.Printf("Warning: Failed to log event: %v\n", err)
    }

    fmt.Printf("Rule '%s' successfully updated.\n", name)
}

func showRule(name string) {
    file, err := os.Open(configFile)
    if err != nil {
        fmt.Println("Failed to open the configuration file:", err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    found := false
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, name+" = ") {
            fmt.Println(line)
            found = true
            break
        }
    }

    if !found {
        fmt.Printf("Rule '%s' does not exist.\n", name)
    }
}

func runCommands(commands []string, bottleValues map[string]string) {
    var processedCommands []string
    for _, cmd := range commands {
        rule, err := getCommand(cmd)
        if err != nil {
            fmt.Printf("Error: %s\n", err)
            continue
        }
        processedRule := processBottles(rule, bottleValues)
        processedCommands = append(processedCommands, processedRule)
    }
    if len(processedCommands) == 0 {
        fmt.Println("No rules found to execute.")
        return
    }
    for i, command := range processedCommands {
        start := time.Now()
        fmt.Printf("Executing command %d: %s\n", i+1, command)
        err := executeCommand(command)
        duration := time.Since(start)

        result := "Success"
        if err != nil {
            result = fmt.Sprintf("Error: %v", err)
            fmt.Printf("Error executing command %d: %s\n", i+1, err)
        }

        logDetails := fmt.Sprintf("Command: \"%s\", Result: %s in %v", command, result, duration)
        err = logEvent("EXECUTE_RULE", logDetails)
        if err != nil {
            fmt.Printf("Warning: Failed to log event: %v\n", err)
        }
    }
}

func getCommand(name string) (string, error) {
    file, err := os.Open(configFile)
    if err != nil {
        return "", fmt.Errorf("failed to open the configuration file: %v", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, name+" = ") {
            return strings.TrimSpace(strings.TrimPrefix(line, name+" = ")), nil
        }
    }

    if err := scanner.Err(); err != nil {
        return "", fmt.Errorf("error reading the configuration file: %v", err)
    }

    return "", fmt.Errorf("rule '%s' not found", name)
}

func importRulesFromFile(filePath string) {
    // Start timing
    start := time.Now()

    // Open the file
    file, err := os.Open(filePath)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // Read the file
    scanner := bufio.NewScanner(file)
    var rulesText string
    for scanner.Scan() {
        rulesText += scanner.Text() + "\n"
    }
    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading file:", err)
        return
    }

    // Extract rules from the text
    rules := extractRules(rulesText)

    // Read existing rules from the configuration file
    existingRules, err := readLines(configFile)
    if err != nil {
        fmt.Println("Error reading existing rules:", err)
        return
    }

    // Process each rule
    for _, rule := range rules {
        parts := strings.Split(rule, " = ")
        if len(parts) != 2 {
            fmt.Println("Error parsing rule:", rule)
            continue
        }
        name := strings.TrimSpace(parts[0])
        command := strings.TrimSpace(parts[1])

        // Check if the rule already exists
        exists := false
        for i, existingRule := range existingRules {
            if strings.HasPrefix(existingRule, name+" = ") {
                exists = true
                fmt.Printf("Rule '%s' already exists. Do you want to overwrite it? (y/n): ", name)
                var response string
                fmt.Scanln(&response)
                if response == "y" {
                    existingRules[i] = fmt.Sprintf("%s = %s", name, command)
                    fmt.Printf("Rule '%s' updated.\n", name)
                } else {
                    fmt.Printf("Skipping rule '%s'.\n", name)
                }
                break
            }
        }

        if !exists {
            existingRules = append(existingRules, fmt.Sprintf("%s = %s", name, command))
            fmt.Printf("Rule '%s' added.\n", name)
        }

        // Create the script immediately
        scriptPath := filepath.Join(filepath.Join(os.Getenv("HOME"), ".local/bin"), name)
        scriptContent := fmt.Sprintf(`#!/bin/bash
start=$(date +%%s.%%N)
%s
end=$(date +%%s.%%N)
duration=$(echo "$end - $start" | bc)
echo "[$(date +'%%Y-%%m-%%d %%H:%%M:%%S')] EXECUTE_RULE %s at $(hostname -I | awk '{print $1}') | Rule: %s, Command: "%s", Result: Success, Duration: ${duration}s" >> %s
`, command, os.Getenv("USER"), name, command, filepath.Join(os.Getenv("HOME"), logDir, logFileName))

        err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
        if err != nil {
            fmt.Printf("Error creating script for rule %s: %v\n", name, err)
        }

        // Log the import event
        err = logEvent("IMPORT_RULE", fmt.Sprintf("From File: %s, Name: %s, Command: %s", filePath, name, command))
        if err != nil {
            fmt.Printf("Warning: Failed to log event: %v\n", err)
        }
    }

    // Write all rules back to the configuration file
    err = writeLinesWithLock(configFile, existingRules)
    if err != nil {
        fmt.Println("Error writing rules to config file:", err)
        return
    }

    // End timing
    end := time.Now()
    duration := end.Sub(start).Seconds()
    fmt.Printf("Rules imported successfully in %.2f seconds.\n", duration)
}

func createScriptForRule(name, command string) {
    scriptPath := filepath.Join(os.Getenv("HOME"), ".local/bin", name)
    scriptContent := fmt.Sprintf("#!/bin/bash\n%s\n", command)

    err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
    if err != nil {
        fmt.Printf("Error creating script for rule %s: %v\n", name, err)
    }
}

func extractRules(text string) []string {
    var rules []string

    re := regexp.MustCompile(`b:([^=]+) = (.*?):b`)
    matches := re.FindAllStringSubmatch(text, -1)
    for _, match := range matches {
        ruleName := strings.TrimSpace(match[1])
        ruleCommand := strings.TrimSpace(match[2])

        // Replace HTML entities with their actual characters
        ruleCommand = html.UnescapeString(ruleCommand)

        rule := fmt.Sprintf("%s = %s", ruleName, ruleCommand)
        rules = append(rules, rule)
    }

    return rules
}

func exportRules() {
    fmt.Println("Exporting rules in progress... Press ctrl+c to quit")
    fmt.Println("You can export rules in bulk, e.g., <rule1> <rule2>")

    var exportRules []string
    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Println("Which rule(s) do you want to export? Leave blank to export all:")
        scanner.Scan()
        text := scanner.Text()

        if text == "" {
            exportRules = getAllRules()
            break
        } else {
            rules := strings.Fields(text)
            // Reset the exportRules slice for re-selection
			exportRules = nil
            var invalidRules []string
            for _, rule := range rules {
                if ruleExists(rule) {
                    exportRules = append(exportRules, rule)
                } else {
                    invalidRules = append(invalidRules, rule)
                }
            }

            if len(invalidRules) > 0 {
                fmt.Printf("The following rules were not found: %v\n", invalidRules)
                fmt.Println("Please re-enter the correct rules or leave blank to export all.")
            } else {
                break
            }
        }
    }

    if len(exportRules) == 0 {
        fmt.Println("No valid rules selected for export.")
        return
    }

    fmt.Println("Do you want to add a comment? Leave blank to continue:")
    scanner.Scan()
    comment := scanner.Text()

    // Prepare export content
    var exportContent []string
    if comment != "" {
        exportContent = append(exportContent, fmt.Sprintf("#%s", comment))
    }

    for _, rule := range exportRules {
        command, err := getCommand(rule)
        if err != nil {
            fmt.Printf("Error getting command for rule '%s': %v\n", rule, err)
            continue
        }
        exportContent = append(exportContent, fmt.Sprintf("b:%s = %s:b", rule, command))
    }

    for {
        fmt.Println("Where do you want to store your file? Leave blank to store in $HOME")
        fmt.Println("Select a folder for your file:")
        scanner.Scan()
        exportPath := scanner.Text()

        if exportPath == "" {
            exportPath = os.Getenv("HOME")
        }

        // Check if the path is valid
        fileInfo, err := os.Stat(exportPath)
        if err != nil || !fileInfo.IsDir() {
            fmt.Println("Location not found or not a directory.")
            continue
        }

        // Write to file
        exportFilePath := fmt.Sprintf("%s/abbtr-rules.txt", exportPath)
        err = writeToFile(exportFilePath, exportContent)
        if err != nil {
            fmt.Println("Error writing rules to file:", err)
            return
        }

        fmt.Printf("Rules successfully exported to: %s\n", exportFilePath)

        // Log the export event
        exportedRules := strings.Join(exportRules, ", ")
        err = logEvent("EXPORT_RULES", fmt.Sprintf("Exported rules: %s, To file: %s", exportedRules, exportFilePath))
        if err != nil {
        fmt.Printf("Warning: Failed to log event: %v\n", err)
        }

        break
    }
}

func ruleExists(name string) bool {
    file, err := os.Open(configFile)
    if err != nil {
        return false
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, name+" = ") {
            return true
        }
    }

    return false
}

func getAllRules() []string {
    var rules []string

    file, err := os.Open(configFile)
    if err != nil {
        fmt.Println("Failed to open the configuration file:", err)
        return rules
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.SplitN(line, "=", 2)
        if len(parts) == 2 {
            rules = append(rules, strings.TrimSpace(parts[0]))
        }
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading the configuration file:", err)
    }

    return rules
}

func writeToFile(filePath string, content []string) error {
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for _, line := range content {
        _, err := fmt.Fprintln(writer, line)
        if err != nil {
            return fmt.Errorf("failed to write to file: %v", err)
        }
    }
    if err := writer.Flush(); err != nil {
        return fmt.Errorf("failed to flush writer: %v", err)
    }

    return nil
}

func executeCommand(command string) error {
    start := time.Now()

    cmd := exec.Command("bash", "-c", command)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    err := cmd.Run()

    duration := time.Since(start)
    result := "Success"
    if err != nil {
        result = fmt.Sprintf("Error: %v", err)
    }

    logDetails := fmt.Sprintf("Command: \"%s\", Result: %s, Duration: %v", command, result, duration)
    logErr := logEvent("EXECUTE_RULE", logDetails)
    if logErr != nil {
        fmt.Printf("Warning: Failed to log event: %v\n", logErr)
    }

    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            return fmt.Errorf("command failed with exit code %d: %v", exitError.ExitCode(), err)
        }
        return fmt.Errorf("failed to execute command: %v", err)
    }
    return nil
}

func processBottles(command string, bottleValues map[string]string) string {
    re := regexp.MustCompile(`b%\('([^']+)'\)%b`)
    return re.ReplaceAllStringFunc(command, func(match string) string {
        bottleName := re.FindStringSubmatch(match)[1]
        if value, ok := bottleValues[bottleName]; ok {
            return value
        }
        fmt.Printf("The %s is?: ", bottleName)
        var value string
        fmt.Scanln(&value)
        return value
    })
}

func readLines(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return lines, scanner.Err()
}

func writeLines(filename string, lines []string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for _, line := range lines {
        // Write the line after unquoting it to interpret escaped characters
        unquotedLine, err := strconv.Unquote(`"` + line + `"`)
        if err != nil {
            fmt.Println("Error unquoting line:", line, "Error:", err)
            return err
        }
        _, err = fmt.Fprintln(writer, unquotedLine)
        if err != nil {
            return err
        }
    }
    return writer.Flush()
}

func logEvent(eventType, details string) error {
    logPath := filepath.Join(os.Getenv("HOME"), logDir, logFileName)

    err := os.MkdirAll(filepath.Dir(logPath), 0755)
    if err != nil {
        return fmt.Errorf("failed to create log directory: %v", err)
    }

    file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("failed to open log file: %v", err)
    }
    defer file.Close()

    user := os.Getenv("USER")
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    ip := getIP()

    logMessage := fmt.Sprintf("[%s] %s %s at %s | %s\n", // i.e User:%s
                              timestamp, eventType, user, ip, details)

    _, err = file.WriteString(logMessage)
    if err != nil {
        return fmt.Errorf("failed to write to log file: %v", err)
    }

    return nil
}

func initConfigFile() error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get home directory: %v", err)
    }

    configDirPath := filepath.Join(homeDir, configDir)
    configPath := filepath.Join(configDirPath, configFileName)

    // Create the directory if it doesn't exist
    err = os.MkdirAll(configDirPath, 0755)
    if err != nil {
        return fmt.Errorf("failed to create config directory: %v", err)
    }

    // Open or create the configuration file with read-write permissions for the user
    file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        if os.IsPermission(err) {
            log.Printf("Permission error: %v\n", err)
            return fmt.Errorf("failed to open or create config file due to permissions: %v", err)
        }
        log.Printf("Failed to open or create config file: %v\n", err)
        return fmt.Errorf("failed to open or create config file: %v", err)
    }
    defer file.Close()

    return nil
}

func getIP() string {
    addrs, err := net.InterfaceAddrs()
    if err == nil {
        for _, addr := range addrs {
            if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
                if ipnet.IP.To4() != nil {
                    return ipnet.IP.String()
                }
            }
        }
    }
    return "Unknown IP"
}

func isReservedName(name string) bool {
    for _, reserved := range reservedNames {
        if name == reserved {
            return true
        }
    }
    return false
}

func writeLinesWithLock(filename string, lines []string) error {
    file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for _, line := range lines {
        _, err = fmt.Fprintln(writer, line)
        if err != nil {
            return err
        }
    }
    return writer.Flush()
}

func syncRulesWithScripts() error {
    rules := getAllRules()
    rulesDir := filepath.Join(os.Getenv("HOME"), ".local/bin")

    // Create the directory if it doesn't exist
    err := os.MkdirAll(rulesDir, 0755)
    if err != nil {
        return fmt.Errorf("failed to create rules directory: %v", err)
    }

    // Remove scripts that don't have corresponding rules
    files, err := os.ReadDir(rulesDir)
    if err != nil {
        return fmt.Errorf("failed to read rules directory: %v", err)
    }

    for _, file := range files {
        if !file.IsDir() {
            found := false
            for _, rule := range rules {
                if rule == file.Name() {
                    found = true
                    break
                }
            }
            if !found {
                scriptPath := filepath.Join(rulesDir, file.Name())
                err := os.Remove(scriptPath)
                if err != nil {
                    fmt.Printf("Error deleting orphaned script %s: %v\n", file.Name(), err)
                }
            }
        }
    }

    // Create or update scripts for existing rules
    for _, rule := range rules {
        command, err := getCommand(rule)
        if err != nil {
            fmt.Printf("Error getting command for rule %s: %v\n", rule, err)
            continue
        }

        scriptPath := filepath.Join(rulesDir, rule)
        scriptContent := fmt.Sprintf(`#!/bin/bash
start=$(date +%%s.%%N)
%s
end=$(date +%%s.%%N)
duration=$(echo "$end - $start" | bc)
echo "[$(date +'%%Y-%%m-%%d %%H:%%M:%%S')] EXECUTE_RULE %s at $(hostname -I | awk '{print $1}') | Rule: %s, Command: "%s", Result: Success, Duration: ${duration}s" >> %s
`, command, os.Getenv("USER"), rule, command, filepath.Join(os.Getenv("HOME"), logDir, logFileName))

        err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
        if err != nil {
            fmt.Printf("Error creating/updating script for rule %s: %v\n", rule, err)
        }
    }

    return nil
}

func checkPath() {
    path := os.Getenv("PATH")
    localBin := filepath.Join(os.Getenv("HOME"), ".local/bin")

    // Check if ~/.local/bin is already in PATH
    if strings.Contains(path, localBin) {
        return // Do nothing if it's already in PATH
    }

    // Warn the user if ~/.local/bin is not in PATH
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Printf("~/.local/bin is not in your PATH, do you want to add it? This is necessary to locally run your rules (y/n): ")
        response, _ := reader.ReadString('\n')
        response = strings.TrimSpace(strings.ToLower(response))

        if response == "y" {
            // Add ~/.local/bin to the PATH and update the profile file
            fmt.Println("Adding ~/.local/bin to your PATH...")

            // Determine the shell profile file based on the user's shell
            shell := os.Getenv("SHELL")
            var profileFile string

            if strings.Contains(shell, "bash") {
                profileFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
            } else if strings.Contains(shell, "zsh") {
                profileFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
            } else if strings.Contains(shell, "fish") {
                profileFile = filepath.Join(os.Getenv("HOME"), ".config/fish/config.fish")
            } else {
                // Default to .profile for unknown shells
                profileFile = filepath.Join(os.Getenv("HOME"), ".profile")
            }

            // Append the export command to the profile file
            f, err := os.OpenFile(profileFile, os.O_APPEND|os.O_WRONLY, 0644)
            if err != nil {
                fmt.Printf("Error opening profile file: %v\n", err)
                return
            }
            defer f.Close()

            if _, err = f.WriteString(fmt.Sprintf("\nexport PATH=%s:$PATH\n", localBin)); err != nil {
                fmt.Printf("Error writing to profile file: %v\n", err)
                return
            }

            fmt.Printf("~/.local/bin has been added to your PATH. Please restart your terminal or run 'source %s' to apply the changes.\n", profileFile)
            break

        } else if response == "n" {
            fmt.Println("You can continue using abbtr, but your rules will not run.")
            break

        } else {
            fmt.Println("Please type your option (y/n):")
        }
    }
}