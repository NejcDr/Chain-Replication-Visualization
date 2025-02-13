# Visualization of Chain Replication

Visual Demonstration of Chain Replication. 

Consists of Backend and Frontend part. 

Backend - Golang code that simulates Chain Replication process. 

Frontend - React web app that visualy demonstrates Chain Replication process.

## Requirements
- Downloaded Go Programming Language. (https://go.dev/)
- Downloaded Node.js v16. Will not work with later releases! (https://nodejs.org/en)

## Usage
```bash
# clone a repo
git clone https://github.com/NejcDr/Chain_Algorithm

# Start backend
cd Backend

go run . [-n <number>] [-f <filename>]
# -n - Starting number of servers in chain. Must be a number between 1 and 7. Default value is 5.
# -f - File containing initial servers storage. Must be .txt file. Values must be like <key> | <value> | <version> | <user (red or blue)>. For examples look init.txt.

# Start frontend
cd Frontend

# only first time
npm install

npm start
```
