#!/bin/zsh

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GITHUB_TOKEN is not set."
  exit 1
fi

# Check if GITLAB_TOKEN is set
if [ -z "$GITLAB_TOKEN" ]; then
  echo "Error: GITLAB_TOKEN is not set."
  exit 1
fi

# Run all scripts in the vhs folder
for script in ./*.vhs; do
  # Remove ~/work/demo directory if the script name starts with git-clone
  if [[ $(basename "$script") == git-clone* ]]; then
    if [ -d "$HOME/work/demo" ]; then
      rm -rf "$HOME/work/demo/*"
      echo "Removed directory: $HOME/work/demo"
    fi
  fi

  echo "Running script: $script"
  vhs "$script"
done