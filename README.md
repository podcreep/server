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

Go 1.17 is required to run the server.

## Running locally

In order to run locally, you need to have a postgresql database set up and running. Any decently
modern version should do.

### Environment variables and running

Next, we use a couple of environment variable to configure the database connection, debug mode and
admin secret password.

Finally, run the server. But make sure the environment variable above are visible to it!

    $ go run main.go

There's a helper script that makes running locally a bit easier:

    $ python3 run.py

## Running a client

See either [android/README.md](https://github.com/podcreep/android/blob/master/README.md) or
[web/README.md](https://github.com/podcreep/web/blob/master/README.md) for instructions on running
the clients.
