# Firestore CLI

Firestore is a command line utility to facilitate operations with Firestore from the command line.

## Usage

Currently both `PROJECT_ID` and `GOOGLE_APPLICATION_CREDENTIALS` are required.

```
export PROJECT_ID=my-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# copying a document
firestore document cp /my-collection/my-document /my-collection/another-document

# querying documents in a collection
firestore -p demo-flux collection query --sort firstName --direction desc -f firstName==Vince -f lastName=Petersen /data
```

## Supported Features

### Documents

- [x] delete
- [x] move
- [x] copy
- [x] download
- [x] add

### Collections

- [x] copy
- [x] delete
- [x] list
- [ ] download
- [ ] upload
- [x] query

## Firestore Emulator

To use this tool with the Firestore Emulator you must set the `FIRESTORE_EMULATOR_HOST` environment variable to the appropriate value. For example:

```
export FIRESTORE_EMULATOR_HOST=localhost:9090
```
