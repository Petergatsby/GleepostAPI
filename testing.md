##How to run the Gleepost API locally 

###1. Set up your Go workspace
Download Go here: http://golang.org/doc/install

Set up your local workspace following these instructions:
http://golang.org/doc/code.html

Specifically, you need a ~/go folder in (eg) your home directory, and you need to set $GOPATH in your ~/.profile so the `go` command will always know where to instal packages. 

###2. Get the GleepostAPI repository
If you have installed Go and set $GOPATH correctly, you should just be able to type `go get github.com/draaglom/GleepostAPI` and it will be installed to your $GOPATH.

You may need to install version control systems for some dependencies (mercurial, bzr). On OSX you can install them with Homebrew.

###3. Get the external dependencies
The API requires an instance of MySQL and Redis to run. You can install these with homebrew.

###4. Initialize the database
There is an up to date db structure available at `GleepostAPI/lib/db/example.sql`. 

###5. Edit your configuration file
There is a blank config file at `GleepostAPI/lib/conf.json`; copy that into the /GleepostAPI/ directory and set the appropriate variables for your installation of MySQL and Redis.

###6. Run the tests!
`go test .`. If that fails, contact me because something is missing from this guide.
