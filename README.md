# CheckpointDrive
A tool to let you back up and sync game files with google drive (designed for linux)

# How to install and use (prebuild)
Download the [Latest Release](https://github.com/GuySandler/CheckpointDrive/releases/tag/Build1)

add to bin:
```bash
sudo mv /path/to/cpd /usr/local/bin/
sudo chmod +x /usr/local/bin/cpd
```

connect drive
```bash 
cpd config drive
```

# Build
1. Go get your google cloud oauth credentials and put it under /pkg/gdrive as credentials.json

2. compile:
```bash
go build -o cpd main.go
```
3. move to bin:
```bash
sudo mv cpd /usr/local/bin/
```

# Commands
## add
add game to the sync list
```bash
cpd add [path to file/directory] [name]
```

### optional flags:
config individual time to sync a spesific game in minutes
```bash
-i 60
```
exclude from daemon (no auto sync)
```bash
-n
```

## remove
remove game from the sync list
```bash
cpd remove [name]
```

## daemon
handle the daemon to automatically sync (run just runs it once if you want to not sync the ignored files)
```bash
cpd daemon [start, stop, restart, status, run]
```

## list
list all games that are in the sync list
```bash
cpd list
```

## sync
manually sync all games on the list
```bash
cpd sync
```

manually sync spesific games separated by commas
```bash
cpd sync game1,game2
```

## config
edit the config file (using nano)
```bash
cpd config edit
```

connect a google account (required to run at least once)
```bash
cpd config drive
```
config the global default time between auto-syncs
```bash
cpd config set daemon-interval 60
```