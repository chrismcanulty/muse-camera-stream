#!/usr/bin/env bash

# Uninstall script for AgentDVR/ Linux/ OSX
# To execute: save and `chmod +x ./agent-uninstall-service.sh` then `./agent-uninstall-service.sh`

if [[ ("$OSTYPE" == "darwin"*) ]]; then
  sudo launchctl unload -w /Library/LaunchDaemons/com.ispy.agent.dvr.plist
  if [ -f /Library/LaunchDaemons/com.ispy.agent.dvr.plist ]; then
    sudo rm -f /Library/LaunchDaemons/com.ispy.agent.dvr.plist
  fi
else
  sudo systemctl stop AgentDVR.service
  sudo systemctl disable AgentDVR.service
  sudo rm /etc/systemd/system/AgentDVR.service
  sudo systemctl daemon-reload
  sudo systemctl reset-failed
fi

echo "stopped and removed service"