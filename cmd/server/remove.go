package main

import "os"

func removeSocket() error {
	return os.RemoveAll("/tmp/engine.sock")
}