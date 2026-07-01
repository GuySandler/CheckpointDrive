# CheckpointDrive

A tool to let you back up and sync game files with google drive

# build
1. Go get your google cloud oauth credentials and put it under /pkg/gdrive as credentials.json

2. compile:
```bash
go build -o cpd main.go
```
3. move to bin:
```bash
sudo mv cpd /usr/local/bin/
```