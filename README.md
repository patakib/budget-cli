# Surprisingly Simple Budget Tracking App
This is a simple budget tracking command-line application written in Go.

## Installation

Prerequisite: Go has to be installed.

- clone repository
- ```go install```
- copy ```config.yaml``` nearby your installed binary or create a custom configuration with the same structure.

## Usage
- create new budget based on your config: ```budget create```
- add new expense: ```budget add -d YYYY-MM-DD -c groceries -a 10000 -m "random message"```
- get monthly status: ```budget status```
