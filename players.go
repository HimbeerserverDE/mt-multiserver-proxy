package main

import "sync"

var players = make(map[string]struct{})
var playersMu sync.RWMutex
