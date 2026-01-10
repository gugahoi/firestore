package browse

import (
	"log"
	"os"
)

var debugLog *log.Logger
var debugFile *os.File

func init() {
	// Create a debug log file
	f, err := os.OpenFile("/tmp/firestore-browse-debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	debugFile = f
	debugLog = log.New(f, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func logDebug(format string, args ...interface{}) {
	debugLog.Printf(format, args...)
	debugFile.Sync() // Flush immediately
}

func logPanic(location string, err interface{}) {
	debugLog.Printf("PANIC at %s: %v", location, err)
	debugFile.Sync() // Flush immediately
}
