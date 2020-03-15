# server
The podcreep backend server and API.

## Getting Started

Check out the three repositories. Ideally you'll want them in the same root directory, so something
like this:

    $ mkdir podcreep && cd podcreep
    $ git clone git@github.com:podcreep/server.git
    $ git clone git@github.com:podcreep/web.git
    $ git clone git@github.com:podcreep/android.git

## Dependencies

Go 1.12 is required to run the server.

## Running locally

This requires a bit of set up to run locally, unfortunately. The latest versions of App Engine don't
really have very good support for locally running.

First, you need to get the Cloud datastore emulator running. That's 

    $ gcloud beta emulators datastore start --data-dir=<path-to-data> \
         --host-port=localhost:12783 --project=<projectid>

The `--data-dir` you specify doesn't really matter, it's just a good idea to specify something so
that you can have consistent data between runs.

`--project` should be set to an App Engine project ID (I use `podcreep` as that's the project ID of
the main server).

Next, you have to set a bunch of environment variables so that the client library can run the
emulator. The variables you have to set can be discovered by running:

    $ gcloud beta emulators datastore env-init --data-dir=<path-to-data>

The `--data-dir` has to be the same as you specified when running the emulator (the emulator has
to be running for this command to work). The environemnt variables don't change so as long as you
specify the same host port and project ID when starting the emulator, you can fairly easily script
it all.

Finally, run the server. But make sure the environment variable above are visible to it!

    $ go run main.go

## Running a client

See either [android/README.md](https://github.com/podcreep/android/blob/master/README.md) or
[web/README.md](https://github.com/podcreep/web/blob/master/README.md) for instructions on running
the clients.
