@echo off
REM Command Generation Script for Claude Code Spec Workflow (Windows)
REM
REM This script generates individual task commands for each task in a spec's tasks.md file.
REM It creates a folder structure under .claude/commands/{spec-name}/ with individual
REM command files for each task that call /spec-execute with the appropriate parameters.
REM
REM Usage: generate-commands.bat <spec-name>

setlocal enabledelayedexpansion

if "%~1"=="" (
    echo Error: Spec name is required
    echo Usage: generate-commands.bat ^<spec-name^>
    exit /b 1
)

set "SPEC_NAME=%~1"
set "PROJECT_ROOT=%CD%"
set "SPEC_DIR=%PROJECT_ROOT%\.claude\specs\%SPEC_NAME%"
set "TASKS_FILE=%SPEC_DIR%\tasks.md"
set "COMMANDS_SPEC_DIR=%PROJECT_ROOT%\.claude\commands\%SPEC_NAME%"

REM Check if tasks.md exists
if not exist "%TASKS_FILE%" (
    echo Error: tasks.md not found at %TASKS_FILE%
    exit /b 1
)

REM Create spec commands directory
if not exist "%COMMANDS_SPEC_DIR%" mkdir "%COMMANDS_SPEC_DIR%"

REM Parse tasks and generate commands
set "TASK_COUNT=0"
set "CURRENT_TASK_ID="
set "CURRENT_TASK_DESC="
set "CURRENT_LEVERAGE="
set "CURRENT_REQUIREMENTS="
set "IN_TASK=0"
echo Parsing tasks from %TASKS_FILE%...

for /f "usebackq delims=" %%a in ("%TASKS_FILE%") do (
    set "LINE=%%a"
    call :ParseTaskLine "!LINE!"
)

REM Handle the last task if we were in one
if !IN_TASK!==1 (
    if not "!CURRENT_TASK_ID!"=="" (
        call :GenerateTaskCommand "!CURRENT_TASK_ID!" "!CURRENT_TASK_DESC!" "!CURRENT_LEVERAGE!" "!CURRENT_REQUIREMENTS!"
        set /a TASK_COUNT+=1
    )
)

echo.
echo Generated !TASK_COUNT! task commands for spec: %SPEC_NAME%
echo Commands created in: .claude\commands\%SPEC_NAME%\
echo.
echo Generated commands:
for /f "usebackq delims=" %%a in ("%TASKS_FILE%") do (
    set "LINE=%%a"
    call :ShowTaskCommand "!LINE!"
)
echo.
echo ============================================================
echo IMPORTANT: Please restart Claude Code for the new commands to be visible
echo ============================================================
echo.
echo The task commands have been generated successfully.
echo After restarting Claude Code, you can use commands like:
echo   /%SPEC_NAME%-task-1
echo   /%SPEC_NAME%-task-2
echo   etc.
echo.

goto :eof

:ParseTaskLine
set "TASK_LINE=%~1"
REM Match task lines like "- [ ] 1. Task description" or "- [ ] 2.1 Task description"
echo !TASK_LINE! | findstr /b /c:"- [ ]" >nul
if !errorlevel! equ 0 (
    REM If we were processing a previous task, generate its command first
    if !IN_TASK!==1 (
        if not "!CURRENT_TASK_ID!"=="" (
            call :GenerateTaskCommand "!CURRENT_TASK_ID!" "!CURRENT_TASK_DESC!" "!CURRENT_LEVERAGE!" "!CURRENT_REQUIREMENTS!"
            set /a TASK_COUNT+=1
        )
    )
    
    REM Extract everything after "- [ ] "
    set "AFTER_CHECKBOX=!TASK_LINE:~6!"
    call :ExtractTaskInfo "!AFTER_CHECKBOX!"
    set "IN_TASK=1"
    set "CURRENT_LEVERAGE="
    set "CURRENT_REQUIREMENTS="
) else if !IN_TASK!==1 (
    REM Check for _Leverage: lines
    echo !TASK_LINE! | findstr /c:"_Leverage:" >nul
    if !errorlevel! equ 0 (
        for /f "tokens=1,* delims=:" %%x in ("!TASK_LINE!") do (
            set "LEVERAGE_PART=%%y"
            REM Remove leading spaces and dashes
            set "LEVERAGE_PART=!LEVERAGE_PART:~1!"
            set "CURRENT_LEVERAGE=!LEVERAGE_PART!"
        )
    )
    
    REM Check for _Requirements: lines
    echo !TASK_LINE! | findstr /c:"_Requirements:" >nul
    if !errorlevel! equ 0 (
        for /f "tokens=1,* delims=:" %%x in ("!TASK_LINE!") do (
            set "REQUIREMENTS_PART=%%y"
            REM Remove leading spaces and dashes
            set "REQUIREMENTS_PART=!REQUIREMENTS_PART:~1!"
            set "CURRENT_REQUIREMENTS=!REQUIREMENTS_PART!"
        )
    )
)
goto :eof

:ExtractTaskInfo
set "REMAINING=%~1"
REM Find the first token which should be the task ID
for /f "tokens=1,* delims= " %%x in ("!REMAINING!") do (
    set "POTENTIAL_ID=%%x"
    set "REST=%%y"

    REM Remove trailing dot if present
    if "!POTENTIAL_ID:~-1!"=="." set "POTENTIAL_ID=!POTENTIAL_ID:~0,-1!"

    REM Check if it looks like a task ID (starts with digit, may contain dots)
    echo !POTENTIAL_ID! | findstr /r "^[0-9]" >nul
    if !errorlevel! equ 0 (
        REM Simple validation - check if it matches pattern like 1, 1.2, 1.2.3 etc
        REM Replace dots with spaces and check each part is numeric
        set "ID_PARTS=!POTENTIAL_ID:.= !"
        set "VALID_ID=1"
        for %%p in (!ID_PARTS!) do (
            REM Use arithmetic test to check if part is numeric
            set /a "TEST_NUM=%%p" 2>nul
            if !errorlevel! neq 0 set "VALID_ID=0"
        )
        if !VALID_ID! equ 1 (
            set "CURRENT_TASK_ID=!POTENTIAL_ID!"
            set "CURRENT_TASK_DESC=!REST!"
        )
    )
)
goto :eof

:GenerateTaskCommand
set "TASK_ID=%~1"
set "TASK_DESC=%~2"
set "LEVERAGE_INFO=%~3"
set "REQUIREMENTS_INFO=%~4"
set "COMMAND_FILE=%COMMANDS_SPEC_DIR%\task-%TASK_ID%.md"

(
echo # %SPEC_NAME% - Task %TASK_ID%
echo.
echo Execute task %TASK_ID% for the %SPEC_NAME% specification.
echo.
echo ## Task Description
echo %TASK_DESC%
echo.
if not "%LEVERAGE_INFO%"=="" (
    echo ## Code Reuse
    echo **Leverage existing code**: %LEVERAGE_INFO%
    echo.
)
if not "%REQUIREMENTS_INFO%"=="" (
    echo ## Requirements Reference  
    echo **Requirements**: %REQUIREMENTS_INFO%
    echo.
)
echo ## Usage
echo ```
echo /%SPEC_NAME%-task-%TASK_ID%
echo ```
echo.
echo ## Instructions
echo This command executes a specific task from the %SPEC_NAME% specification.
echo.
echo **Automatic Execution**: This command will automatically execute:
echo ```
echo /spec-execute %TASK_ID% %SPEC_NAME%
echo ```
echo.
echo **Process**:
echo 1. Load the %SPEC_NAME% specification context ^(requirements.md, design.md, tasks.md^)
echo 2. Execute task %TASK_ID%: "%TASK_DESC%"
echo 3. **Prioritize code reuse**: Use existing components and utilities identified above
echo 4. Follow all implementation guidelines from the main /spec-execute command
echo 5. Mark the task as complete in tasks.md
echo 6. Stop and wait for user review
echo.
echo **Important**: This command follows the same rules as /spec-execute:
echo - Execute ONLY this specific task
echo - **Leverage existing code** whenever possible to avoid rebuilding functionality
echo - Mark task as complete by changing [ ] to [x] in tasks.md
echo - Stop after completion and wait for user approval
echo - Do not automatically proceed to the next task
echo.
echo ## Next Steps
echo After task completion, you can:
echo - Review the implementation
echo - Run tests if applicable
echo - Execute the next task using /%SPEC_NAME%-task-[next-id]
echo - Check overall progress with /spec-status %SPEC_NAME%
) > "%COMMAND_FILE%"

goto :eof

:ShowTaskCommand
set "TASK_LINE=%~1"
REM Use same logic as ParseTaskLine
echo !TASK_LINE! | findstr /b /c:"- [ ]" >nul
if !errorlevel! equ 0 (
    set "AFTER_CHECKBOX=!TASK_LINE:~6!"
    call :ShowTaskInfo "!AFTER_CHECKBOX!"
)
goto :eof

:ShowTaskInfo
set "REMAINING=%~1"
for /f "tokens=1,* delims= " %%x in ("!REMAINING!") do (
    set "POTENTIAL_ID=%%x"
    set "REST=%%y"

    if "!POTENTIAL_ID:~-1!"=="." set "POTENTIAL_ID=!POTENTIAL_ID:~0,-1!"

    REM Check if it looks like a task ID
    echo !POTENTIAL_ID! | findstr /r "^[0-9]" >nul
    if !errorlevel! equ 0 (
        REM Simple validation - check if it matches pattern like 1, 1.2, 1.2.3 etc
        set "ID_PARTS=!POTENTIAL_ID:.= !"
        set "VALID_ID=1"
        for %%p in (!ID_PARTS!) do (
            REM Use arithmetic test to check if part is numeric
            set /a "TEST_NUM=%%p" 2>nul
            if !errorlevel! neq 0 set "VALID_ID=0"
        )
        if !VALID_ID! equ 1 (
            echo   /%SPEC_NAME%-task-!POTENTIAL_ID! - !REST!
        )
    )
)
goto :eof
