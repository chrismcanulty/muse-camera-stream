#!/usr/bin/env bash


# Get the directory where the script is located
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

# Define log file path in the same directory as the script
LOG_FILE="$SCRIPT_DIR/agent-update.log"

# Clear the log file at the start
> "$LOG_FILE"

# Attempt to redirect both stdout and stderr to the log file
if ! exec > >(tee -a "$LOG_FILE") 2>&1; then
    echo "Warning: Failed to set up logging to $LOG_FILE" >&2
    # Optionally, you can decide to set up alternative logging here
fi

isOSX=0
if [[ ("$OSTYPE" == "darwin"*) ]]; then
	isOSX=1
fi

svc=0

if [ $isOSX -eq 1 ]; then
	if [ -f /Library/LaunchDaemons/com.ispy.agent.dvr.plist ] || [ -f ~/Library/LaunchAgents/com.ispy.agent.dvr.plist ]; then
        svc=0
        echo "Service is installed on macOS"
    else
        svc=1
        echo "Service not installed on macOS"
    fi
else
	if systemctl is-enabled --quiet AgentDVR.service; then
        svc=0
        echo "Service found on Linux"
    else
        svc=1
        echo "Service not found on Linux"
    fi
fi

# Ensure unzip is installed
if command -v apt-get >/dev/null; then
    apt-get update && apt-get install -y unzip
elif command -v yum >/dev/null; then
    yum install -y unzip
fi

if [ "$1" == "-update" ]; then
	if [ -f AgentDVR.zip ]; then
		echo "Installing update"
		unzip -o AgentDVR.zip
		rm -f AgentDVR.zip

		chmod +x "$SCRIPT_DIR/Agent"
		find . -name "*.sh" -exec chmod +x {} \;
	else
		echo "Update file not found. Use the web portal to update Agent"
	fi
fi

# Install plugins
if [ "$1" == "-plugins" ]; then 
    if [ -d "Plugins/$2" ]; then
        cd Plugins
        cd "$2"
        if [ -f plugin.zip ]; then
            unzip -o plugin.zip
            echo "Unzipped"
            rm -f plugin.zip
            echo "Removed archive"
            cd ..
        else
            echo "Plugin archive not found"
        fi
        cd ..
    else
        echo "Plugins directory not found or incorrect path: Plugins/$2"
        exit 1
    fi
fi

# Restart or start the service with error handling
if [ $isOSX -eq 1 ]; then
    if [ $svc -eq 0 ]; then
        echo "Restarting Agent service on macOS"
        launchctl stop "com.ispy.agent.dvr"
        if [ $? -ne 0 ]; then
            echo "Error stopping service, starting Agent manually. Error code: $?"
            "$SCRIPT_DIR/Agent"
        else
            launchctl start "com.ispy.agent.dvr"
            if [ $? -ne 0 ]; then
                echo "Error starting service, starting Agent manually. Error code: $?"
                "$SCRIPT_DIR/Agent"
            fi
        fi
    else
        echo "Service not found, starting Agent manually on macOS"
        "$SCRIPT_DIR/Agent"
    fi
else
    if [ $svc -eq 0 ]; then
        echo "Restarting Agent service on Linux"
        systemctl restart AgentDVR.service
        if [ $? -ne 0 ]; then
            echo "Error restarting service, starting Agent manually. Error code: $?"
            "$SCRIPT_DIR/Agent"
        fi
    else
        echo "Service not found, starting Agent manually on Linux"
        "$SCRIPT_DIR/Agent"
    fi
fi
