#!/bin/bash

# If you have a phone connected for running the android app, this will allow it to connect to
# our server. It's OK if this call fails, you might need to run it manually if you connect a
# phone at a later time.
adb reverse tcp:8080 tcp:8080

# Datastore properties that the client library uses to connect to the local datastore emulator.
# Make sure you set up the emulator to run on the port listed below.
set DATASTORE_DATASET=podcreep
set DATASTORE_EMULATOR_HOST=localhost:12783
set DATASTORE_EMULATOR_HOST_PATH=localhost:12783/datastore
set DATASTORE_HOST=http://localhost:12783
set DATASTORE_PROJECT_ID=pod-creep

# Let the server know it's running in debug mode.
set DEBUG=1

# Put your App engine service credential in this file so we can authenticate with Google's APIs
# (used for admin login only at the moment).
set GOOGLE_APPLICATION_CREDENTIALS=/Users/deanh/src/podcreep/service-credentials.json

go run main.go
