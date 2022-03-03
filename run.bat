@echo off

rem If you have a phone connected for running the android app, this will allow it to connect to
rem our server. It's OK if this call fails, you might need to run it manually if you connect a
rem phone at a later time.
adb reverse tcp:8080 tcp:8080

rem URL for the database. This should be in the form
rem "postgres://username:password@localhost:5432/database_name"
rem Before running, do `SET DBPASSWD=<password>` (or $Env:DBPASSWD="password" for powershell) to set
rem the correct password
set DATABASE_URL=postgres://podcreep_user:%DBPASSWD%@localhost/podcreep

rem Let the server know it's running in debug mode.
set DEBUG=1

rem This is the password used to access the backend admin pages.
set ADMIN_PASSWORD=secret

go run main.go
