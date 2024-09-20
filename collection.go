package main

import (
	"log"
	"os"

	"cloud.google.com/go/firestore"
)

// Collection is used to process commands related to collection actions.
type Collection struct {
	client *firestore.Client
}

func NewCollection(client *firestore.Client) Collection {
	return Collection{client}
}

func (c Collection) usage() {
	log.SetOutput(os.Stderr)
	log.Fatalln(`
Description:
	Perform actions on firestore collections.
Usage: 
	firestore collection [action] <...args>
Actions:
	cp - copy a collection from the source to the destination, recursively
	rm - delete a collection
	ls - list document ids in the collection
Examples: 
	firestore collection cp /source/collection/path /destination/collection/path
	firestore collection rm /path/to/collection
	firestore collection ls /path/to/collection`)

}
func (c Collection) checkArgs(args []string, size int) {
	if len(args) < size {
		c.usage()
	}
}

// func (c Collection) Run(args []string) error {
// 	c.checkArgs(args, 1)
// 	var err error
// 	action := args[0]
//
// 	switch action {
// 	case "cp":
// 		c.checkArgs(args, 3)
// 		src := args[1]
// 		dst := args[2]
// 		err = c.copy(src, dst)
// 	case "ls":
// 		c.checkArgs(args, 2)
// 		src := args[1]
// 		err = c.ls(src)
// 	case "rm":
// 		c.checkArgs(args, 2)
// 		src := args[1]
// 		err = c.rm(src)
// 	default:
// 		return fmt.Errorf("action not found, available: cp")
// 	}
//
// 	return err
// }
