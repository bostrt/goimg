package main

type Config struct {
	bind       string
	data       string
	db         string
	gcInterval int // Seconds, default 300s
	gcLimit    int // Number of entries to scan each gc, default 100
}
