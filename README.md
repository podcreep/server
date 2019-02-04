# server
The podcreep backend server and API.

## Getting Started

Check out the three repositories. Ideally you'll want them in the same root directory, so something like this:

    $ mkdir podcreep && cd podcreep
    $ git clone git@github.com:podcreep/server.git
    $ git clone git@github.com:podcreep/web.git
    $ git clone git@github.com:podcreep/android.git

## Dependencies

Go 1.11 is required to run the server.

## Running locally

    $ dev_appserver.py <path-to>\server --enable_host_checking 0

The `--enable_host_checking 0` flag is to allow you to connect to the server via localhost, 127.0.0.1, or your machine's hostname. It's entirely optional.

## Running a client

See either [android/README.md](https://github.com/podcreep/android/blob/master/README.md) or [web/README.md](https://github.com/podcreep/web/blob/master/README.md) for instructions on running the clients.
