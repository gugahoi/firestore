# Firestore CLI

Firestore is a command line utility to facilitate operations with Firestore from the command line.

## Usage

Currently both `PROJECT_ID` and `GOOGLE_APPLICATION_CREDENTIALS` are required.

```
export PROJECT_ID=my-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
firestore document cp /my-collection/my-document /my-collection/another-document
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
- [ ] query

## Firestore Emulator

To use this tool with the Firestore Emulator you must set the `FIRESTORE_EMULATOR_HOST` environment variable to the appropriate value. For example:

```
export FIRESTORE_EMULATOR_HOST=localhost:9090
```
